package jsonw

/*
#include <stdlib.h>
#include "jsonw.h"
*/
import "C"

import (
	"encoding/json" // borda-json: o valor vem do desenvolvedor de fora e vai à ABI em JSON
	"errors"
	"unsafe"
)

// Indexar monta o índice do campo neste leitor (troca o de outro campo).
func (l *Leitor) Indexar(campo int) error {
	if C.jsonw_indexar(l.a, C.int(campo)) != 0 {
		return errors.New("jsonw: campo sem indice")
	}
	return nil
}

// TemIndice diz se o campo tem índice neste leitor.
func (l *Leitor) TemIndice(campo int) bool { return C.jsonw_tem_indice(l.a, C.int(campo)) == 1 }

// Chave devolve, em ordem crescente, as linhas em que o campo vale o valor (7, "RJ", nil).
// Com Indexar, a busca é binária; sem índice, varre o arquivo.
func (l *Leitor) Chave(campo int, valor any) ([]uint64, error) {
	v, err := json.Marshal(valor)
	if err != nil {
		return nil, err
	}
	cv, livre := cstr(string(v))
	defer livre()
	out := make([]uint64, 16)
	for {
		n := int64(C.jsonw_chave(l.a, C.int(campo), cv, C.size_t(len(v)),
			(*C.uint64_t)(unsafe.Pointer(unsafe.SliceData(out))), C.size_t(len(out))))
		switch {
		case n == -2:
			return nil, ErrTipo
		case n < 0:
			return nil, errors.New("jsonw: campo invalido")
		case n <= int64(len(out)):
			return out[:n], nil
		}
		out = make([]uint64, n)
	}
}
