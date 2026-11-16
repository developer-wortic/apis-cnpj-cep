#!/bin/sh
# consultar.sh: consulta um CNPJ e um CEP nas APIs abertas da Wortic. O curl baixa o registro em jsonW binario
# (Accept: application/jsonw); o wjsw confere a custodia e le o registro.
#
#   ./consultar.sh 00000000000191 01310100
#
# Precisa do curl e do wjsw (o programa de linha de comando do jsonW) no PATH.
set -eu

CNPJ=${1:-00000000000191}
CEP=${2:-01310100}

# So digitos, no tamanho certo: o valor vai para a URL.
case "$CNPJ" in
  *[!0-9]*|'') echo "CNPJ precisa ter so digitos" >&2; exit 2 ;;
esac
case "$CEP" in
  *[!0-9]*|'') echo "CEP precisa ter so digitos" >&2; exit 2 ;;
esac

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -s -f -H 'Accept: application/jsonw' "https://cnpj.wortic.com/cnpj/$CNPJ" -o "$TMP/cnpj.jsw"
curl -s -f -H 'Accept: application/jsonw' "https://cep.wortic.com/cep/$CEP" -o "$TMP/cep.jsw"

echo "== CNPJ $CNPJ"
wjsw conferir "$TMP/cnpj.jsw"
wjsw ver "$TMP/cnpj.jsw" 0 1

echo "== CEP $CEP"
wjsw conferir "$TMP/cep.jsw"
wjsw ver "$TMP/cep.jsw" 0 1
