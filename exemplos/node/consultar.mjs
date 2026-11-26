// consultar.mjs: consulta um CNPJ e um CEP nas APIs abertas da Wortic em jsonW binario e le o registro na hora.
//
//   node consultar.mjs 00000000000191 01310100
//
// (1) baixa a resposta com fetch (Accept: application/jsonw); (2) abre os bytes com o SDK em WebAssembly
// (sdk/jsonw-node.js): sem addon nativo e sem arquivo temporario; (3) confere a custodia com conferir();
// (4) le o registro com linha(0).
import { carregar } from "./sdk/jsonw-node.js";

const URL_CNPJ = "https://cnpj.wortic.com/cnpj/";
const URL_CEP = "https://cep.wortic.com/cep/";

const soDigitos = (s, n) => s.length === n && /^[0-9]+$/.test(s);

// Baixa a resposta da API em jsonW binario.
async function baixar(url) {
  const resp = await fetch(url, { headers: { Accept: "application/jsonw" } });
  if (!resp.ok) throw new Error(`HTTP ${resp.status} em ${url}`);
  return new Uint8Array(await resp.arrayBuffer());
}

// Abre os bytes e confere a custodia (conferir() lanca erro se algum bloco nao confere).
async function abrir(j, url) {
  const a = j.abrir(await baixar(url));
  a.conferir();
  return a;
}

async function consultarCnpj(j, cnpj) {
  const a = await abrir(j, URL_CNPJ + cnpj);
  try {
    const r = a.linha(0);
    // JSON.parse converteria o decimal em numero e perderia o formato; o texto cru do campo sai do JSON da linha.
    const capital = /"capital_social":([^,}]+)/.exec(a.json(0))?.[1] ?? "-";
    console.log("CNPJ", cnpj);
    console.log("razao social:", r.razao_social);
    console.log(`situacao: ${r.situacao.descricao} (desde ${r.situacao.desde})`);
    console.log("capital social:", capital);
    console.log("abertura:", r.abertura);
    console.log("custodia: ok");
  } finally {
    a.fechar();
  }
}

async function consultarCep(j, cep) {
  const a = await abrir(j, URL_CEP + cep);
  try {
    const r = a.linha(0);
    console.log("CEP", cep);
    console.log("logradouro:", r.logradouro);
    console.log(`cidade: ${r.cidade}/${r.uf}`);
    console.log("custodia: ok");
  } finally {
    a.fechar();
  }
}

const [cnpj = "00000000000191", cep = "01310100"] = process.argv.slice(2);
if (!soDigitos(cnpj, 14) || !soDigitos(cep, 8)) {
  console.error("uso: node consultar.mjs <cnpj com 14 digitos> <cep com 8 digitos>");
  process.exit(2);
}

const j = await carregar();
let falhou = 0;
for (const [rotulo, fn, valor] of [
  ["CNPJ", consultarCnpj, cnpj],
  ["CEP", consultarCep, cep],
]) {
  try {
    await fn(j, valor);
  } catch (e) {
    console.log(rotulo, valor, "falhou:", e.message);
    falhou++;
  }
}
process.exitCode = falhou ? 1 : 0;
