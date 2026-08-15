-- SGF-Rondonópolis, Fase 6: adequação ao trecho da IN SCL Nº 04/2021
-- específico de contratos de MÃO DE OBRA TERCEIRIZADA (não todo contrato
-- de serviço — a IN04 é mais estreita que TipoObjeto=SERVICO). Ver a
-- Matriz Normativa (.claude/plans/projeto-selene-rippling-kite.md).
--
-- Camada 2 (regra do SGF, não da norma): o enum TipoObjeto existente
-- (CONSUMO/PERMANENTE/SERVICO) não distingue terceirização de mão de
-- obra de outros serviços — este flag marca isso explicitamente.
ALTER TABLE contratos ADD COLUMN exige_fiscalizacao_terceirizacao boolean NOT NULL DEFAULT false;

-- Documentos mensais específicos de mão de obra terceirizada (IN04
-- Art.9º-XXXII, alíneas a/b.1/b.2/b.3) — não fazem parte do checklist
-- genérico de nenhum outro tipo de contrato, só entram quando
-- exige_fiscalizacao_terceirizacao=true (ver internal/service/checklist.go).
-- Numa base nova, database.Seed() já cria estes registros a partir de
-- tiposDocumentoSeed antes desta migration rodar de novo — ON CONFLICT
-- evita erro de duplicidade nesse caso; em base já existente, é esta
-- migration que garante o backfill.
INSERT INTO tipos_documento (nome, exige_validade) VALUES
    ('Comprovante de Pagamento de Salário', false),
    ('Protocolo GFIP', false),
    ('Guia GRF/GPS', false),
    ('Relação de Trabalhadores (SEFIP)', false)
ON CONFLICT (nome) DO NOTHING;
