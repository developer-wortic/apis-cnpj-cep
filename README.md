# APIs abertas de CNPJ e CEP da Wortic, em JSON e em jsonW

Duas APIs grátis, sem login e sem chave:

- **CNPJ** (`https://cnpj.wortic.com`): a ficha da Receita Federal de 73,8 milhões de estabelecimentos, os sócios (documento mascarado) e a busca de empresa pelo nome.
- **CEP** (`https://cep.wortic.com`): o endereço pelo CEP (base dos Correios) e a busca do CEP pela rua.

Limite do grátis: 60 consultas por minuto por IP. O plano pago usa o cabeçalho `x-api-key` e tem limite maior.
Documentação: [cnpj.wortic.com/docs](https://cnpj.wortic.com/docs) e [cep.wortic.com/docs](https://cep.wortic.com/docs).

| Rota | O que devolve |
|---|---|
| `GET https://cnpj.wortic.com/cnpj/{cnpj}` | a ficha (razão social, situação, abertura, CNAEs, natureza, capital, porte, Simples, MEI, endereço) |
| `GET https://cnpj.wortic.com/cnpj/{cnpj}/socios` | os sócios |
| `GET https://cnpj.wortic.com/empresas?nome=..&uf=..` | empresas ativas pelo nome |
| `GET https://cep.wortic.com/cep/{cep}` | o endereço do CEP |
| `GET https://cep.wortic.com/cep/buscar?uf=..&cidade=..&logradouro=..&limite=..` | os CEPs de uma rua |

## JSON ou jsonW: você escolhe

JSON continua sendo o padrão. Quem pede jsonW recebe **o mesmo conteúdo**, com o esquema tipado da rota.

| Você pede | Recebe |
|---|---|
| nada, `?formato=json` | JSON (`application/json`) |
| `?formato=jsonw-texto` ou `Accept: text/jsonw` | jsonW-texto numa linha (`text/jsonw`); o esquema vem no cabeçalho `x-jsonw-esquema` |
| `?formato=jsonw` (ou `jsw`) ou `Accept: application/jsonw` | o arquivo `.jsw` de um registro (`application/vnd.wortic.jsonw`) |

```
curl 'https://cnpj.wortic.com/cnpj/00000000000191?formato=jsonw-texto'
{cnpj:"00000000000191",...,situacao:{codigo:2,descricao:"Ativa",desde:t"2005-11-03T00:00:00.000000Z"},abertura:t"1966-08-01T00:00:00.000000Z",...,capital_social:120000000000.00d,...}

curl -H 'Accept: application/jsonw' https://cnpj.wortic.com/cnpj/00000000000191 -o bb.jsw
wjsw conferir bb.jsw    # ok: 1 linhas, 1 blocos, sha256 e crc32c conferidos
```

O que o jsonW traz que o JSON não traz:

- **Tipos de verdade.** Data sai como `tempo` (`t"2005-11-03T00:00:00.000000Z"`), não como texto. O capital social sai como decimal exato (`120000000000.00d`), sem passar por ponto flutuante: no JSON ele chega como `120000000000.0`.
- **Custódia.** O `.jsw` traz o hash de cada bloco (sha256) e o crc32c. O seu programa confere com `conferir()` que a resposta chegou inteira e sem alteração.
- **O mesmo formato para guardar.** Se você guarda as consultas, um arquivo `.jsw` é colunar, com índice e leitura direta de um campo, sem subir banco de dados. Os SDKs, em mais de dez linguagens, estão em [developer-wortic/jsonw](https://github.com/developer-wortic/jsonw).

### Tamanho, medido em 16/11/2026 (bytes)

| Resposta | JSON | JSON com gzip | jsonW-texto | jsonW-texto com gzip | `.jsw` |
|---|---:|---:|---:|---:|---:|
| ficha do CNPJ 00000000000191 | 1.072 | 637 | 1.031 | 643 | 2.388 |
| sócios do CNPJ 00000000000191 (lista) | 10.271 | 1.141 | 10.022 | 1.145 | 2.116 |
| CEP 01310100 | 222 | 177 | 203 | 176 | 1.092 |
| CEPs da Av. Paulista em São Paulo, 20 (lista) | 4.906 | 636 | 4.503 | 628 | 1.284 |

Leitura honesta: o jsonW-texto é de 4% a 9% menor que o JSON cru. O `.jsw` é **74% a 79% menor que o JSON cru nas listas** (colunar, com dicionário), mas num registro só ele é maior, porque leva o índice e o hash da custódia. Comprimido com gzip, o JSON das listas ainda fica menor que o `.jsw`. Use o `.jsw` quando quiser o tipo, a custódia e um arquivo pronto para guardar e consultar. Para tráfego mínimo de um registro, use o JSON com compressão (a borda já entrega com brotli ou gzip quando o seu cliente aceita).

## Velocidade, medida em 16/11/2026

Método (o mesmo antes e depois): `curl` de dentro da nossa máquina em São Paulo, tempo total da chamada (DNS, TCP, TLS, espera e corpo), 15 entradas diferentes, 1 segundo entre as chamadas. A mesma entrada para nós e para a [BrasilAPI](https://brasilapi.com.br). Mediana. O roteiro está em [`medicao/medir.sh`](medicao/medir.sh) e cada chamada está em `medicao/*.jsw.txt`.

| | Antes (16/11, tarde) | Depois, entrada nova (ninguém tinha em cache) | Depois, a mesma entrada de novo (cache) |
|---|---:|---:|---:|
| Wortic CNPJ | 0,10 s | **0,071 s** (achou 15 de 15) | **0,046 s** |
| BrasilAPI CNPJ | 0,03 a 0,05 s (5 entradas) | 0,416 s (achou 0 de 15) | 0,042 s (o 404 em cache) |
| Wortic CEP | 0,09 s | **0,058 s** (15 de 15) | **0,040 s** |
| BrasilAPI CEP | 0,03 a 0,05 s (5 entradas) | 2,455 s (13 de 15) | 0,042 s |

Dentro do servidor, sem rede (pedido HTTP local, mesmas 300 entradas sorteadas da base, medido antes e depois na mesma máquina):

| | Antes | Depois |
|---|---:|---:|
| Ficha do CNPJ, leitura quente | 15,2 ms | **0,72 ms** |
| Ficha do CNPJ, leitura fria (região do arquivo nunca lida) | 22,4 ms | 7,6 ms |
| Sócios | 7,6 ms | **0,46 ms** |
| CEP | 4,0 ms | **0,46 ms** |
| Aperto de mão TLS na borda (handshake, do cliente) | 38 ms | **12 ms** |

As 300 respostas de cada rota saíram **byte a byte iguais** às de antes (CNPJ, sócios e CEP; a única diferença é o cabeçalho novo `vary: accept`).

O que ainda não dá: passando pela Cloudflare no plano grátis, a resposta mais rápida possível medida daqui (TCP + TLS + espera da borda, mesmo com cache) fica perto de 0,04 s. A BrasilAPI está atrás da mesma borda e bate no mesmo piso. Abaixo de 0,03 s só sem a Cloudflare no caminho. Medição de um ponto só, num dia só. Não é o tempo que o seu navegador vê.

### O que mudou

1. **A busca acha o bloco pela zona, sem descomprimir.** O arquivo jsonW guarda, por bloco, os 8 primeiros bytes do menor e do maior valor de cada texto. Como a base está ordenada pelo CNPJ, a busca vai direto ao bloco certo. Antes ela descomprimia uns 27 pedaços do arquivo por consulta.
2. **Uma cópia da base feita para consulta pontual.** A base da Receita foi regravada com blocos de 16 KiB (`wjsw regravar --bloco 16384`), no lugar de 1 a 8 MiB. Ler um CNPJ abre só o pedaço daquele CNPJ. A base de blocos grandes continua sendo a das varreduras. A cópia se refaz sozinha quando a base do mês muda.
3. **Certificado com cadeia curta.** A cadeia do certificado da borda passou para a do Google Trust Services. A verificação no cliente caiu de cerca de 30 ms para 3 ms (OpenSSL 4).
4. **Cache na borda** para as consultas grátis (1 dia para o que achou, 1 hora para o que não achou). Consulta com `x-api-key` ou com jsonW no `Accept` não entra no cache.

## Exemplos de cliente lendo jsonW

Cada pasta tem o exemplo e a `PROVA.txt` com o comando e a saída real, rodados contra as APIs no ar em 16/11/2026.

| Linguagem | Exemplo | Como lê o `.jsw` |
|---|---|---|
| C | [`exemplos/c/consultar.c`](exemplos/c/consultar.c) | `jsonw_abrir_memoria`, direto dos bytes |
| Go | [`exemplos/go/consultar.go`](exemplos/go/consultar.go) | `jsonw.Abrir` sobre um arquivo temporário |
| Rust | [`exemplos/rust/src/main.rs`](exemplos/rust/src/main.rs) | dos bytes, pela ABI |
| Node.js | [`exemplos/node/consultar.mjs`](exemplos/node/consultar.mjs) | SDK WASM, `abrir_memoria` |
| PHP 8 (FFI) | [`exemplos/php/consultar.php`](exemplos/php/consultar.php) | `JsonW\Leitor` sobre um arquivo temporário |
| Shell | [`exemplos/shell/consultar.sh`](exemplos/shell/consultar.sh) | `curl` + `wjsw ver` e `wjsw conferir` |

Saída de todos, para o CNPJ 00000000000191 e o CEP 01310100:

```
razao social: BANCO DO BRASIL SA
capital social: 120000000000.00
custodia: ok
logradouro: Avenida Paulista
cidade: São Paulo/SP
```

## Licença

MIT. Os dados vêm da Receita Federal (dados abertos do CNPJ) e dos Correios, pela Wortic.
