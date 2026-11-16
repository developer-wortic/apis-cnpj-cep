<?php
/* consultar.php: consulta um CNPJ e um CEP nas APIs abertas da Wortic em jsonW binario e le o registro na hora.
 *
 *   php consultar.php 00000000000191 01310100
 *
 * (1) baixa a resposta com file_get_contents (Accept: application/jsonw); (2) grava num .jsw temporario, porque o
 * Leitor do SDK abre por caminho; (3) confere a custodia com conferir(); (4) le o registro com linha() e json().
 * Precisa da extensao FFI (PHP 8.4+; o SDK carrega a libjsonw.so da pasta sdk/).
 */
declare(strict_types=1);

require __DIR__ . '/sdk/jsonw.php';

const URL_CNPJ = 'https://cnpj.wortic.com/cnpj/';
const URL_CEP = 'https://cep.wortic.com/cep/';

// Baixa a resposta da API em jsonW binario.
function baixar(string $url): string
{
    $ctx = stream_context_create(['http' => ['header' => "Accept: application/jsonw\r\n", 'timeout' => 20]]);
    $corpo = @file_get_contents($url, false, $ctx);
    if ($corpo === false || $corpo === '') {
        throw new RuntimeException("a API nao respondeu ($url)");
    }
    return $corpo;
}

// Grava o registro num .jsw temporario, abre com o SDK e confere a custodia.
function abrir(string $url, string $temp): JsonW\Leitor
{
    file_put_contents($temp, baixar($url));
    $leitor = new JsonW\Leitor($temp);
    $leitor->conferir();
    return $leitor;
}

function consultarCnpj(string $cnpj): void
{
    $temp = tempnam(sys_get_temp_dir(), 'consulta-cnpj-');
    try {
        $l = abrir(URL_CNPJ . $cnpj, $temp);
        $r = $l->linha(0);
        // json_decode transformaria o decimal em float e perderia o formato; o texto cru do campo sai do JSON da linha.
        preg_match('/"capital_social":([^,}]+)/', $l->json(0), $m);
        echo "CNPJ $cnpj\n";
        echo 'razao social: ', $r['razao_social'], "\n";
        echo "situacao: {$r['situacao']['descricao']} (desde {$r['situacao']['desde']})\n";
        echo 'capital social: ', $m[1] ?? '-', "\n";
        echo 'abertura: ', $r['abertura'], "\n";
        echo "custodia: ok\n";
        $l->fechar();
    } finally {
        @unlink($temp);
    }
}

function consultarCep(string $cep): void
{
    $temp = tempnam(sys_get_temp_dir(), 'consulta-cep-');
    try {
        $l = abrir(URL_CEP . $cep, $temp);
        $r = $l->linha(0);
        echo "CEP $cep\n";
        echo 'logradouro: ', $r['logradouro'], "\n";
        echo "cidade: {$r['cidade']}/{$r['uf']}\n";
        echo "custodia: ok\n";
        $l->fechar();
    } finally {
        @unlink($temp);
    }
}

$cnpj = $argv[1] ?? '00000000000191';
$cep = $argv[2] ?? '01310100';
if (!preg_match('/^\d{14}$/', $cnpj) || !preg_match('/^\d{8}$/', $cep)) {
    fwrite(STDERR, "uso: php consultar.php <cnpj com 14 digitos> <cep com 8 digitos>\n");
    exit(2);
}

$falhou = 0;
foreach ([['CNPJ', $cnpj, 'consultarCnpj'], ['CEP', $cep, 'consultarCep']] as [$rotulo, $valor, $fn]) {
    try {
        $fn($valor);
    } catch (Throwable $e) {
        echo "$rotulo $valor falhou: ", $e->getMessage(), "\n";
        $falhou++;
    }
}
exit($falhou ? 1 : 0);
