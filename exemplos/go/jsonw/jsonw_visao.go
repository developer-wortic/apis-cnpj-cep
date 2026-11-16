package jsonw

/*
#include <stdlib.h>
#include "jsonw.h"
*/
import "C"

// Visao trava no leitor o que a sessão vê ("pessoal vendas tenant:42"). Depois dela, Esquema e Campos são só os das
// colunas visíveis, a linha de outro tenant devolve ErrOculta e AvaliarJSON recusa campo escondido. Não destrava.
func (l *Leitor) Visao(escopos string) error {
	var e erroC
	c, livre := cstr(escopos)
	defer livre()
	if C.jsonw_visao(l.a, c, e.ptr(), e.tam()) != 0 {
		return e.Err()
	}
	l.lerEsquema()
	return nil
}
