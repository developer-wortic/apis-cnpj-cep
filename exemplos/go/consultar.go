// consultar.go: consulta um CNPJ e um CEP nas APIs abertas da Wortic em jsonW binario e le o registro na hora.
//
//	go build -o consultar . && ./consultar 00000000000191 01310100
//
// (1) baixa a resposta com net/http (Accept: application/jsonw); (2) grava num .jsw temporario, porque o Leitor
// do SDK abre por caminho; (3) confere a custodia com Conferir; (4) le o registro com Linha.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"jsonw"
)

const (
	urlCNPJ = "https://cnpj.wortic.com/cnpj/"
	urlCEP  = "https://cep.wortic.com/cep/"
)

var soDigitos = regexp.MustCompile(`^[0-9]+$`)

// baixar pega a resposta da API em jsonW binario.
func baixar(url string) ([]byte, error) {
	cli := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/jsonw")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// abrir baixa o registro, grava num .jsw temporario, abre com o SDK e confere a custodia.
// Devolve o leitor e a funcao que fecha e apaga o arquivo temporario.
func abrir(url string) (*jsonw.Leitor, func(), error) {
	corpo, err := baixar(url)
	if err != nil {
		return nil, nil, err
	}
	arq, err := os.CreateTemp("", "consulta-*.jsw")
	if err != nil {
		return nil, nil, err
	}
	nome := arq.Name()
	_, err = arq.Write(corpo)
	if cerr := arq.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(nome)
		return nil, nil, err
	}
	l, err := jsonw.Abrir(nome)
	if err != nil {
		os.Remove(nome)
		return nil, nil, err
	}
	if err := l.Conferir(); err != nil {
		l.Fechar()
		os.Remove(nome)
		return nil, nil, err
	}
	return l, func() { l.Fechar(); os.Remove(nome) }, nil
}

type situacao struct {
	Descricao string `json:"descricao"`
	Desde     string `json:"desde"`
}

type cnpjRegistro struct {
	RazaoSocial   string      `json:"razao_social"`
	Situacao      situacao    `json:"situacao"`
	CapitalSocial json.Number `json:"capital_social"` // decimal[2]: o texto exato, sem passar por float
	Abertura      string      `json:"abertura"`
}

type cepRegistro struct {
	Logradouro string `json:"logradouro"`
	Cidade     string `json:"cidade"`
	UF         string `json:"uf"`
}

func consultarCNPJ(cnpj string) error {
	l, fechar, err := abrir(urlCNPJ + cnpj)
	if err != nil {
		return err
	}
	defer fechar()
	var r cnpjRegistro
	if err := l.Linha(0, &r); err != nil {
		return err
	}
	fmt.Println("CNPJ", cnpj)
	fmt.Println("razao social:", r.RazaoSocial)
	fmt.Printf("situacao: %s (desde %s)\n", r.Situacao.Descricao, r.Situacao.Desde)
	fmt.Println("capital social:", r.CapitalSocial)
	fmt.Println("abertura:", r.Abertura)
	fmt.Println("custodia: ok")
	return nil
}

func consultarCEP(cep string) error {
	l, fechar, err := abrir(urlCEP + cep)
	if err != nil {
		return err
	}
	defer fechar()
	var r cepRegistro
	if err := l.Linha(0, &r); err != nil {
		return err
	}
	fmt.Println("CEP", cep)
	fmt.Println("logradouro:", r.Logradouro)
	fmt.Printf("cidade: %s/%s\n", r.Cidade, r.UF)
	fmt.Println("custodia: ok")
	return nil
}

func main() {
	cnpj, cep := "00000000000191", "01310100"
	if len(os.Args) > 1 {
		cnpj = os.Args[1]
	}
	if len(os.Args) > 2 {
		cep = os.Args[2]
	}
	if len(cnpj) != 14 || !soDigitos.MatchString(cnpj) || len(cep) != 8 || !soDigitos.MatchString(cep) {
		fmt.Fprintln(os.Stderr, "uso: consultar <cnpj com 14 digitos> <cep com 8 digitos>")
		os.Exit(2)
	}
	falhou := 0
	if err := consultarCNPJ(cnpj); err != nil {
		fmt.Println("CNPJ", cnpj, "falhou:", err)
		falhou++
	}
	if err := consultarCEP(cep); err != nil {
		fmt.Println("CEP", cep, "falhou:", err)
		falhou++
	}
	if falhou > 0 {
		os.Exit(1)
	}
}
