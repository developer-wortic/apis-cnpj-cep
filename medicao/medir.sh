#!/bin/bash
# medir.sh <rotulo>: o método do cmo (16/11/2026): curl de dentro da v3, tempo total, 15 entradas distintas, 1 s entre chamadas.
# Mesma entrada para a Wortic e a BrasilAPI. Saída: uma linha por chamada em medicao-<rotulo>.jsw.txt e a mediana de cada uma.
cd "$(dirname "$0")"; R=${1:-agora}; S=medicao-$R.jsw.txt; : > $S
med(){ sort -n | awk '{a[NR]=$1} END{printf "%.3f", a[int((NR+1)/2)]}'; }
um(){ curl -s -o /dev/null -w "%{http_code} %{time_total} %{time_appconnect} %{time_starttransfer}" "$1"; }
for api in wortic-cnpj brasil-cnpj wortic-cep brasil-cep; do
  case $api in *cnpj) L=cnpjs15.txt;; *cep) L=ceps15.txt;; esac
  while read x; do case $api in wortic-cnpj) u=https://cnpj.wortic.com/cnpj/$x;; brasil-cnpj) u=https://brasilapi.com.br/api/cnpj/v1/$x;; wortic-cep) u=https://cep.wortic.com/cep/$x;; brasil-cep) u=https://brasilapi.com.br/api/cep/v2/$x;; esac
    set -- $(um "$u"); echo "{rodada:\"$R\",api:\"$api\",entrada:\"$x\",status:$1,total_s:$2,tls_s:$3,ttfb_s:$4}" >> $S; sleep 1; done < $L
  echo "$api: mediana total $(grep "api:\"$api\"" $S | grep -oE 'total_s:[0-9.]+' | cut -d: -f2 | med) s | status 200: $(grep "api:\"$api\"" $S | grep -c 'status:200')/15"
done
