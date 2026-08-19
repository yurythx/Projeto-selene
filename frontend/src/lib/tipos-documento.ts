import type { TipoDocumento } from "@/lib/api/client";

/**
 * Espelha service.TipoDocumentoAplicavel no backend (ver
 * internal/service/checklist.go) — mesma regra, sem duplicar as listas
 * de nomes: RestritoTipoObjeto/RestritoTerceirizacao já vêm prontos do
 * backend em cada TipoDocumento (dados, não lógica de negócio). Usado
 * pra filtrar o select de upload em ProcessoDialog conforme o
 * TipoObjeto/ExigeFiscalizacaoTerceirizacao do contrato do processo —
 * o backend (DocumentoService.Upload) é quem faz valer a regra de
 * verdade; este filtro é só pra não oferecer no select uma opção que
 * o backend rejeitaria (ex: "Boleto DAM" num contrato de CONSUMO).
 */
export function tipoDocumentoAplicavel(
  tipo: TipoDocumento,
  tipoObjeto: string | undefined,
  exigeFiscalizacaoTerceirizacao: boolean | undefined
): boolean {
  if (tipo.RestritoTipoObjeto && tipo.RestritoTipoObjeto !== tipoObjeto) {
    return false;
  }
  if (tipo.RestritoTerceirizacao && !exigeFiscalizacaoTerceirizacao) {
    return false;
  }
  return true;
}

/** Filtra a lista completa de tipos de documento para os aplicáveis a um contrato específico. */
export function tiposDocumentoAplicaveis(
  tipos: TipoDocumento[],
  tipoObjeto: string | undefined,
  exigeFiscalizacaoTerceirizacao: boolean | undefined
): TipoDocumento[] {
  return tipos.filter((tipo) => tipoDocumentoAplicavel(tipo, tipoObjeto, exigeFiscalizacaoTerceirizacao));
}
