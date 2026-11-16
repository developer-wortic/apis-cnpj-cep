package jsonw

/*
#include <stdlib.h>
#include "jsonw.h"
*/
import "C"

import (
	"encoding/json"
	"unsafe"
)

// Nota é uma linha escolhida e a nota dela.
type Nota struct {
	Linha uint64  `json:"linha"`
	Nota  float64 `json:"nota"`
}

// AvaliarJSON roda o pedido ({"pesos":..} | {"cosseno":..} | {"esparso":..} | {"grade":..}) e devolve as k melhores em JSON.
func (l *Leitor) AvaliarJSON(pedido string, k int) (string, error) {
	var e erroC
	p, livre := cstr(pedido)
	defer livre()
	s, err := texto(func(b *C.char, n C.size_t) C.int64_t {
		return C.jsonw_avaliar(l.a, p, C.size_t(len(pedido)), C.size_t(k), b, n, e.ptr(), e.tam())
	})
	if err != nil && e[0] != 0 {
		return "", e.Err()
	}
	return s, err
}

// Avaliar aceita o pedido como texto JSON ou como valor Go e devolve as k melhores.
func (l *Leitor) Avaliar(pedido any, k int) ([]Nota, error) {
	p, ok := pedido.(string)
	if !ok {
		b, err := json.Marshal(pedido)
		if err != nil {
			return nil, err
		}
		p = string(b)
	}
	s, err := l.AvaliarJSON(p, k)
	if err != nil {
		return nil, err
	}
	var r []Nota
	return r, json.Unmarshal([]byte(s), &r)
}

// Escolher devolve as posições das k maiores notas (maior primeiro; empate pela menor posição; NaN é a pior).
func Escolher(notas []float64, k int) []uint64 {
	if len(notas) == 0 || k <= 0 {
		return nil
	}
	out := make([]uint64, k)
	m := C.jsonw_escolher((*C.double)(unsafe.Pointer(unsafe.SliceData(notas))), C.size_t(len(notas)), C.size_t(k),
		(*C.uint64_t)(unsafe.Pointer(unsafe.SliceData(out))))
	if m < 0 {
		return nil
	}
	return out[:m]
}

// Reconhecedor acha os termos de uma tabela num texto e soma os sinais.
type Reconhecedor struct{ r unsafe.Pointer }

// NovoReconhecedor abre a tabela de termos (termo:texto,forma:dic,sinal:dic,peso:f32,idioma:dic,origem:dic).
func NovoReconhecedor(termosJsw string) (*Reconhecedor, error) {
	var e erroC
	c, livre := cstr(termosJsw)
	defer livre()
	r := C.jsonw_reconhecedor(c, e.ptr(), e.tam())
	if r == nil {
		return nil, e.Err()
	}
	return &Reconhecedor{r: r}, nil
}

// Fechar libera o reconhecedor. Chamar de novo não faz nada.
func (r *Reconhecedor) Fechar() {
	if r.r != nil {
		C.jsonw_reconhecedor_fechar(r.r)
		r.r = nil
	}
}

// ReconhecerJSON devolve {"termos":[..],"sinais":[..]} do texto.
func (r *Reconhecedor) ReconhecerJSON(t string) (string, error) {
	c, livre := cstr(t)
	defer livre()
	return texto(func(b *C.char, n C.size_t) C.int64_t { return C.jsonw_reconhecer(r.r, c, C.size_t(len(t)), b, n) })
}

// Reconhecer decodifica o resultado em destino.
func (r *Reconhecedor) Reconhecer(t string, destino any) error {
	s, err := r.ReconhecerJSON(t)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), destino)
}
