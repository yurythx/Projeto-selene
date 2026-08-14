-- Reverte 000001_initial_schema.up.sql — ordem inversa de dependências.
DROP TABLE IF EXISTS kanban_logs;
DROP TABLE IF EXISTS documentos_anexos;
DROP TABLE IF EXISTS processos_pagamento;
DROP TABLE IF EXISTS contratos;
DROP TABLE IF EXISTS tipos_documento;
DROP TABLE IF EXISTS kanban_etapas;
DROP TABLE IF EXISTS users;
