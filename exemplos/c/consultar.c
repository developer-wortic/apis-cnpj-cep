/* consultar.c: consulta um CNPJ e um CEP nas APIs abertas da Wortic em jsonW binario e le o registro na hora.
 *
 *   cc -std=c11 -Wall -Wextra -o consultar consultar.c -Isdk sdk/libjsonw.a -lm -lpthread
 *   ./consultar 00000000000191 01310100
 *
 * (1) baixa a resposta com o curl (Accept: application/jsonw), sem gravar arquivo no disco;
 * (2) abre os bytes na memoria com jsonw_abrir_memoria (ABI C do SDK);
 * (3) confere a custodia com jsonw_conferir (crc32c e hash de cada bloco);
 * (4) le os campos pelo nome com jsonw_campo_json, sem decodificar o registro inteiro.
 */
#define _POSIX_C_SOURCE 200809L
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "jsonw.h"

#define URL_CNPJ "https://cnpj.wortic.com/cnpj/"
#define URL_CEP  "https://cep.wortic.com/cep/"

/* So digitos, no tamanho certo: a entrada vai para a linha do curl, entao nao aceita mais nada. */
static int digitos(const char *s, size_t n) {
    if (strlen(s) != n) return 0;
    for (size_t i = 0; i < n; i++)
        if (s[i] < '0' || s[i] > '9') return 0;
    return 1;
}

/* Baixa a URL com o curl e devolve o corpo (malloc). Devolve 0 se o curl falhou ou veio vazio. */
static int baixar(const char *url, unsigned char **corpo, size_t *tamanho) {
    char cmd[512];
    snprintf(cmd, sizeof cmd, "curl -s -f -H 'Accept: application/jsonw' '%s'", url);
    FILE *p = popen(cmd, "r");
    if (!p) return 0;
    size_t cap = 1 << 14, n = 0, k;
    unsigned char *buf = malloc(cap);
    if (!buf) { pclose(p); return 0; }
    while ((k = fread(buf + n, 1, cap - n, p)) > 0) {
        n += k;
        if (n == cap) {
            unsigned char *nb = realloc(buf, cap * 2);
            if (!nb) { free(buf); pclose(p); return 0; }
            buf = nb;
            cap *= 2;
        }
    }
    if (pclose(p) != 0 || n == 0) { free(buf); return 0; }
    *corpo = buf;
    *tamanho = n;
    return 1;
}

/* Baixa, abre na memoria e confere a custodia. Devolve o leitor, ou NULL com a razao em erro. */
static void *abrir_api(const char *url, char *erro, size_t nerro) {
    unsigned char *corpo;
    size_t n;
    if (!baixar(url, &corpo, &n)) {
        snprintf(erro, nerro, "a API nao respondeu");
        return NULL;
    }
    void *r = jsonw_abrir_memoria(corpo, n, erro, nerro);
    free(corpo);
    if (!r) return NULL;
    if (jsonw_conferir(r, erro, nerro) != 0) {
        jsonw_fechar(r);
        return NULL;
    }
    return r;
}

/* Tira as aspas de um texto JSON ("abc" vira abc). Numero e data passam como estao. */
static void sem_aspas(char *s) {
    size_t n = strlen(s);
    if (n >= 2 && s[0] == '"' && s[n - 1] == '"') {
        memmove(s, s + 1, n - 2);
        s[n - 2] = 0;
    }
}

/* Campo `nome` da linha 0 (sem aspas). Vazio se o esquema nao tem o campo. */
static void campo(void *r, const char *nome, char *saida, size_t cap) {
    saida[0] = 0;
    int total = jsonw_campos(r);
    for (int k = 0; k < total; k++) {
        if (strcmp(jsonw_campo_nome(r, k), nome) == 0) {
            if (jsonw_campo_json(r, 0, k, saida, cap) < 0) saida[0] = 0;
            sem_aspas(saida);
            return;
        }
    }
}

/* Valor simples de uma chave dentro de um objeto JSON ({"codigo":2,"descricao":"Ativa"} -> Ativa). */
static void valor_do_objeto(const char *obj, const char *chave, char *saida, size_t cap) {
    char busca[64];
    snprintf(busca, sizeof busca, "\"%s\":", chave);
    saida[0] = 0;
    const char *p = strstr(obj, busca);
    if (!p) return;
    p += strlen(busca);
    const char *fim = p;
    if (*p == '"') {
        fim = strchr(p + 1, '"');
        fim = fim ? fim + 1 : p;
    } else {
        while (*fim && *fim != ',' && *fim != '}') fim++;
    }
    size_t n = (size_t)(fim - p);
    if (n >= cap) n = cap - 1;
    memcpy(saida, p, n);
    saida[n] = 0;
    sem_aspas(saida);
}

static int consultar_cnpj(const char *cnpj) {
    char url[128], erro[256];
    snprintf(url, sizeof url, "%s%s", URL_CNPJ, cnpj);
    void *r = abrir_api(url, erro, sizeof erro);
    printf("CNPJ %s\n", cnpj);
    if (!r) {
        printf("  falhou: %s\n", erro);
        return 1;
    }
    char razao[256], situacao[512], descricao[128], desde[64], capital[64], abertura[64];
    campo(r, "razao_social", razao, sizeof razao);
    campo(r, "situacao", situacao, sizeof situacao);
    valor_do_objeto(situacao, "descricao", descricao, sizeof descricao);
    valor_do_objeto(situacao, "desde", desde, sizeof desde);
    campo(r, "capital_social", capital, sizeof capital);
    campo(r, "abertura", abertura, sizeof abertura);
    printf("razao social: %s\n", razao);
    printf("situacao: %s (desde %s)\n", descricao, desde);
    printf("capital social: %s\n", capital);
    printf("abertura: %s\n", abertura);
    printf("custodia: ok\n");
    jsonw_fechar(r);
    return 0;
}

static int consultar_cep(const char *cep) {
    char url[128], erro[256];
    snprintf(url, sizeof url, "%s%s", URL_CEP, cep);
    void *r = abrir_api(url, erro, sizeof erro);
    printf("CEP %s\n", cep);
    if (!r) {
        printf("  falhou: %s\n", erro);
        return 1;
    }
    char logradouro[256], cidade[128], uf[8];
    campo(r, "logradouro", logradouro, sizeof logradouro);
    campo(r, "cidade", cidade, sizeof cidade);
    campo(r, "uf", uf, sizeof uf);
    printf("logradouro: %s\n", logradouro);
    printf("cidade: %s/%s\n", cidade, uf);
    printf("custodia: ok\n");
    jsonw_fechar(r);
    return 0;
}

int main(int argc, char **argv) {
    const char *cnpj = argc > 1 ? argv[1] : "00000000000191";
    const char *cep = argc > 2 ? argv[2] : "01310100";
    if (!digitos(cnpj, 14) || !digitos(cep, 8)) {
        fprintf(stderr, "uso: consultar <cnpj com 14 digitos> <cep com 8 digitos>\n");
        return 2;
    }
    int falhou = consultar_cnpj(cnpj);
    falhou += consultar_cep(cep);
    return falhou ? 1 : 0;
}
