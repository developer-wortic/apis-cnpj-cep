// Package jsonw é o jsonW (json Wondrous) para Go, por cima da ABI C (libjsonw.a via cgo).
// Cada linha vai e volta como JSON.
//
//	a, err := jsonw.Abrir("dados.jsw")
//	if err != nil { ... }
//	defer a.Fechar()
//	for i := range a.Linhas {
//		var m map[string]any
//		_ = a.Linha(i, &m)
//	}
//
//	w, _ := jsonw.Criar("novo.jsw", "id:i64,nome:texto")
//	_ = w.Gravar(map[string]any{"id": 1, "nome": "x"})
//	_ = w.Fechar()
package jsonw

/*
#cgo CFLAGS: -I${SRCDIR}/..
#cgo LDFLAGS: ${SRCDIR}/../libjsonw.a -lm
#include <stdlib.h>
#include "jsonw.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"runtime"
	"unsafe"
)

var (
	// ErrOculta: a linha é de outro tenant (depois de Visao); quem itera pula.
	ErrOculta = errors.New("jsonw: linha oculta pela visao")
	// ErrIlegivel: a linha ou o campo não pôde ser lido.
	ErrIlegivel = errors.New("jsonw: linha ou campo ilegivel")
	// ErrTipo: o valor não é do tipo do campo.
	ErrTipo = errors.New("jsonw: valor nao e do tipo do campo")
)

// Leitor lê um arquivo .jsw pelo mmap. Linhas, Esquema e Campos refletem a visão atual.
type Leitor struct {
	a       unsafe.Pointer
	limpeza runtime.Cleanup
	Linhas  uint64
	Esquema string
	Campos  []string
}

// Escritor grava um arquivo .jsw novo; o destino só aparece no Fechar (troca atômica).
type Escritor struct{ w unsafe.Pointer }

// Versao devolve a versão da biblioteca.
func Versao() string { return C.GoString(C.jsonw_versao()) }

// erroC é o buffer de erro que a ABI preenche; Err devolve o texto como error.
type erroC [320]C.char

func (e *erroC) ptr() *C.char         { return &e[0] }
func (e *erroC) tam() C.size_t        { return C.size_t(len(e)) }
func (e *erroC) Err() error           { return errors.New(C.GoString(&e[0])) }
func cstr(s string) (*C.char, func()) { c := C.CString(s); return c, func() { C.free(unsafe.Pointer(c)) } }

// Abrir abre o arquivo para leitura.
func Abrir(caminho string) (*Leitor, error) {
	var e erroC
	c, livre := cstr(caminho)
	defer livre()
	a := C.jsonw_abrir(c, e.ptr(), e.tam())
	if a == nil {
		return nil, e.Err()
	}
	l := &Leitor{a: a}
	l.lerEsquema()
	l.limpeza = runtime.AddCleanup(l, func(a unsafe.Pointer) { C.jsonw_fechar(a) }, a)
	return l, nil
}

func (l *Leitor) lerEsquema() {
	l.Linhas = uint64(C.jsonw_linhas(l.a))
	l.Esquema = C.GoString(C.jsonw_esquema(l.a))
	n := int(C.jsonw_campos(l.a))
	l.Campos = make([]string, 0, n)
	for i := range n {
		l.Campos = append(l.Campos, C.GoString(C.jsonw_campo_nome(l.a, C.int(i))))
	}
}

// Fechar libera o arquivo. Chamar de novo não faz nada.
func (l *Leitor) Fechar() {
	if l.a != nil {
		l.limpeza.Stop()
		C.jsonw_fechar(l.a)
		l.a = nil
	}
}

// texto chama a função da ABI com um buffer que cresce até caber (-2 = curto).
func texto(chamar func(b *C.char, n C.size_t) C.int64_t) (string, error) {
	for tam := 1 << 16; ; tam *= 4 {
		b := make([]byte, tam)
		switch n := chamar((*C.char)(unsafe.Pointer(unsafe.SliceData(b))), C.size_t(tam)); {
		case n == -2:
			continue
		case n == -3:
			return "", ErrOculta
		case n < 0:
			return "", ErrIlegivel
		default:
			return string(b[:n]), nil
		}
	}
}

// JSON devolve a linha i como JSON.
func (l *Leitor) JSON(i uint64) (string, error) {
	return texto(func(b *C.char, n C.size_t) C.int64_t { return C.jsonw_linha_json(l.a, C.uint64_t(i), b, n) })
}

// Texto devolve a linha i em jsonW-texto.
func (l *Leitor) Texto(i uint64) (string, error) {
	return texto(func(b *C.char, n C.size_t) C.int64_t { return C.jsonw_linha_texto(l.a, C.uint64_t(i), b, n) })
}

// Linha decodifica a linha i em destino (como json.Unmarshal).
func (l *Leitor) Linha(i uint64, destino any) error {
	s, err := l.JSON(i)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), destino)
}

// Campo decodifica só o campo k da linha i, sem decodificar o resto.
func (l *Leitor) Campo(i uint64, k int, destino any) error {
	s, err := texto(func(b *C.char, n C.size_t) C.int64_t { return C.jsonw_campo_json(l.a, C.uint64_t(i), C.int(k), b, n) })
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), destino)
}

// Conferir confere o crc32c e o hash de cada bloco e a cadeia do arquivo.
func (l *Leitor) Conferir() error {
	var e erroC
	if C.jsonw_conferir(l.a, e.ptr(), e.tam()) != 0 {
		return e.Err()
	}
	return nil
}

// Criar abre um arquivo novo para gravar no esquema dado.
func Criar(caminho, esquema string) (*Escritor, error) {
	var e erroC
	c, livreC := cstr(caminho)
	defer livreC()
	s, livreS := cstr(esquema)
	defer livreS()
	w := C.jsonw_criar(c, s, e.ptr(), e.tam())
	if w == nil {
		return nil, e.Err()
	}
	return &Escritor{w: w}, nil
}

// Opcoes: compressão 0..2, hash 0 (sha256) ou 1 (blake3), bloco em KiB (0 = 1 MiB).
func (w *Escritor) Opcoes(compressao, hash, blocoKb int) {
	C.jsonw_opcoes(w.w, C.int(compressao), C.int(hash), C.int(blocoKb))
}

// GravarTexto grava uma linha em JSON ou jsonW-texto.
func (w *Escritor) GravarTexto(t string) error {
	var e erroC
	c, livre := cstr(t)
	defer livre()
	if C.jsonw_gravar_json(w.w, c, C.size_t(len(t)), e.ptr(), e.tam()) != 0 {
		return e.Err()
	}
	return nil
}

// Gravar grava obj como uma linha (via json.Marshal).
func (w *Escritor) Gravar(obj any) error {
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return w.GravarTexto(string(b))
}

// Descarregar empurra o bloco em andamento para o disco.
func (w *Escritor) Descarregar() { C.jsonw_descarregar(w.w) }

// Fechar grava o que falta e põe o arquivo no lugar.
func (w *Escritor) Fechar() error {
	if w.w == nil {
		return nil
	}
	r := C.jsonw_fechar_escritor(w.w)
	w.w = nil
	if r != 0 {
		return errors.New("jsonw: nao gravou tudo")
	}
	return nil
}

// Abandonar desiste da gravação: fecha sem publicar, e o destino fica como estava.
func (w *Escritor) Abandonar() {
	if w.w != nil {
		C.jsonw_abandonar(w.w)
		w.w = nil
	}
}
