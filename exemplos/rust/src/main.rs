//! consultar.rs: consulta um CNPJ e um CEP nas APIs abertas da Wortic em jsonW binario e le o registro na hora.
//!
//!   cargo run --release -- 00000000000191 01310100
//!
//! (1) baixa a resposta com o curl (Accept: application/jsonw); (2) grava num .jsw temporario, porque o Leitor do
//! SDK abre por caminho; (3) confere a custodia com conferir(); (4) le os campos pelo nome com campo().
//! So a biblioteca padrao do Rust e o SDK jsonw (sem crate extra).

use std::fs;
use std::path::PathBuf;
use std::process::{Command, ExitCode};

use jsonw::Leitor;

const URL_CNPJ: &str = "https://cnpj.wortic.com/cnpj/";
const URL_CEP: &str = "https://cep.wortic.com/cep/";

fn so_digitos(s: &str, n: usize) -> bool {
    s.len() == n && s.bytes().all(|b| b.is_ascii_digit())
}

/// Tira as aspas de um texto JSON ("abc" vira abc). Numero e data passam como estao.
fn sem_aspas(s: &str) -> &str {
    s.strip_prefix('"').and_then(|t| t.strip_suffix('"')).unwrap_or(s)
}

/// Valor simples de uma chave dentro de um objeto JSON ({"codigo":2,"descricao":"Ativa"} -> Ativa).
fn valor_do_objeto(obj: &str, chave: &str) -> String {
    let busca = format!("\"{chave}\":");
    let Some(i) = obj.find(&busca) else {
        return String::new();
    };
    let resto = &obj[i + busca.len()..];
    let valor = match resto.strip_prefix('"') {
        Some(t) => &t[..t.find('"').unwrap_or(t.len())],
        None => &resto[..resto.find([',', '}']).unwrap_or(resto.len())],
    };
    valor.to_string()
}

/// Um .jsw temporario que some quando sai do escopo.
struct Temporario(PathBuf);

impl Drop for Temporario {
    fn drop(&mut self) {
        let _ = fs::remove_file(&self.0);
    }
}

/// Baixa a resposta da API em jsonW binario com o curl.
fn baixar(url: &str) -> Result<Vec<u8>, String> {
    let saida = Command::new("curl")
        .args(["-s", "-f", "-H", "Accept: application/jsonw", url])
        .output()
        .map_err(|e| format!("curl nao rodou: {e}"))?;
    if !saida.status.success() || saida.stdout.is_empty() {
        return Err(format!("a API nao respondeu ({url})"));
    }
    Ok(saida.stdout)
}

/// Baixa, grava num .jsw temporario, abre com o SDK e confere a custodia.
fn abrir(url: &str, tmp: &Temporario) -> Result<Leitor, String> {
    let corpo = baixar(url)?;
    fs::write(&tmp.0, &corpo).map_err(|e| e.to_string())?;
    let leitor = Leitor::abrir(tmp.0.to_str().unwrap_or("")).map_err(|e| e.to_string())?;
    leitor.conferir().map_err(|e| e.to_string())?;
    Ok(leitor)
}

/// Campo `nome` da linha 0, sem aspas.
fn campo_texto(l: &Leitor, nome: &str) -> Result<String, String> {
    let k = l.indice(nome).map_err(|e| e.to_string())?;
    let bruto = l.campo(0, k).map_err(|e| e.to_string())?;
    Ok(sem_aspas(&bruto).to_string())
}

fn consultar_cnpj(cnpj: &str) -> Result<(), String> {
    let tmp = Temporario(std::env::temp_dir().join(format!("consulta-cnpj-{}.jsw", std::process::id())));
    let l = abrir(&format!("{URL_CNPJ}{cnpj}"), &tmp)?;
    let situacao = l.campo(0, l.indice("situacao").map_err(|e| e.to_string())?).map_err(|e| e.to_string())?;
    // o capital vem como o texto exato do decimal[2] (sem passar por float)
    let capital = l.campo(0, l.indice("capital_social").map_err(|e| e.to_string())?).map_err(|e| e.to_string())?;
    println!("CNPJ {cnpj}");
    println!("razao social: {}", campo_texto(&l, "razao_social")?);
    println!(
        "situacao: {} (desde {})",
        valor_do_objeto(&situacao, "descricao"),
        valor_do_objeto(&situacao, "desde")
    );
    println!("capital social: {capital}");
    println!("abertura: {}", campo_texto(&l, "abertura")?);
    println!("custodia: ok");
    Ok(())
}

fn consultar_cep(cep: &str) -> Result<(), String> {
    let tmp = Temporario(std::env::temp_dir().join(format!("consulta-cep-{}.jsw", std::process::id())));
    let l = abrir(&format!("{URL_CEP}{cep}"), &tmp)?;
    println!("CEP {cep}");
    println!("logradouro: {}", campo_texto(&l, "logradouro")?);
    println!("cidade: {}/{}", campo_texto(&l, "cidade")?, campo_texto(&l, "uf")?);
    println!("custodia: ok");
    Ok(())
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().collect();
    let cnpj = args.get(1).map(String::as_str).unwrap_or("00000000000191");
    let cep = args.get(2).map(String::as_str).unwrap_or("01310100");
    if !so_digitos(cnpj, 14) || !so_digitos(cep, 8) {
        eprintln!("uso: consultar <cnpj com 14 digitos> <cep com 8 digitos>");
        return ExitCode::from(2);
    }
    let mut falhou = 0;
    if let Err(e) = consultar_cnpj(cnpj) {
        println!("CNPJ {cnpj} falhou: {e}");
        falhou += 1;
    }
    if let Err(e) = consultar_cep(cep) {
        println!("CEP {cep} falhou: {e}");
        falhou += 1;
    }
    if falhou > 0 { ExitCode::from(1) } else { ExitCode::SUCCESS }
}
