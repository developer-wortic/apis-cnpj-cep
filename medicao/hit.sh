#!/bin/bash
# hit.sh: espera da borda em cache HIT (mediana de 15), CNPJ e CEP já em cache, mais o cdnjs como referência
med(){ sort -n | awk '{a[NR]=$1} END{printf "%.1f", a[int((NR+1)/2)]*1000}'; }
c=$(sed -n 3p "$(dirname "$0")/cnpjs15.txt"); e=$(sed -n 3p "$(dirname "$0")/ceps15.txt")
for u in https://cnpj.wortic.com/cnpj/$c https://cep.wortic.com/cep/$e https://cdnjs.cloudflare.com/ajax/libs/jquery/3.7.1/jquery.min.js; do
  L=$(for i in $(seq 15); do curl -s -o /dev/null -w "%{time_appconnect} %{time_starttransfer} %{time_total}\n" "$u"; done)
  echo "espera $(echo "$L" | awk '{print $2-$1}' | med) ms | total $(echo "$L" | awk '{print $3}' | med) ms | $u" | cut -c1-110
done
