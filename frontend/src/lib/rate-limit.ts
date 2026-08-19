import "server-only";
import { NextResponse } from "next/server";
import Redis from "ioredis";

/**
 * Defesa em profundidade contra abuso nas mutações do BFF — item do
 * roadmap ("Rate limiting nos Route Handlers do BFF", ver README.md).
 * Redundante com o rate limit por usuário que já existe no backend Go
 * (RATE_LIMIT_RPS/RATE_LIMIT_BURST, `backend/internal/middleware/
 * ratelimit_redis.go`), mas cobre o BFF mesmo se uma rota daqui, por
 * bug, nunca chegasse a acionar o limite do backend (ex.: erro antes do
 * fetch pro backend, ou uma rota que só grava localmente). Compartilha o
 * MESMO Redis do backend (REDIS_ADDR/REDIS_PASSWORD, já provisionado no
 * docker-compose) em vez de subir outro — DB lógico 1 (backend usa o
 * padrão, DB 0), só pra nunca colidir chave por acidente.
 *
 * Chave por IP, não por usuário: todas as 27 rotas de mutação já
 * resolvem a sessão DEPOIS da checagem de origem (assertOrigemSegura),
 * então checar aqui no mesmo ponto (antes da sessão) evita reordenar a
 * lógica interna de cada handler; IP já é o que o backend usa pra
 * requisições sem usuário autenticado no contexto (ver comentário em
 * ratelimit.go: "cai pra ClientIP() na ausência de usuário").
 */

let cliente: Redis | null | undefined;

function getCliente(): Redis | null {
  if (cliente !== undefined) return cliente;

  const addr = process.env.REDIS_ADDR;
  if (!addr) {
    // Sem Redis configurado (ex.: dev local sem o serviço redis rodando)
    // — sem rate limit no BFF, não sem app: fail-open já é a postura
    // desta camada (ver assertDentroDoLimite abaixo).
    cliente = null;
    return cliente;
  }

  const [host, portStr] = addr.split(":");
  cliente = new Redis({
    host,
    port: Number(portStr) || 6379,
    password: process.env.REDIS_PASSWORD,
    db: 1,
    maxRetriesPerRequest: 1,
    lazyConnect: true,
  });
  // Sem listener, uma falha de conexão emitiria um erro não tratado e
  // derrubaria o processo Node — cada chamada de assertDentroDoLimite já
  // trata a falha localmente (try/catch, fail-open), este listener só
  // impede o crash do processo inteiro.
  cliente.on("error", () => {});

  return cliente;
}

const JANELA_SEGUNDOS = 10;
// ~2 req/s sustentado com folga de rajada — mesma ordem de grandeza do
// default do backend (5 rps / burst 10), um pouco mais folgado de
// propósito: esta é a camada REDUNDANTE, quem decide o limite de
// verdade é o backend.
const LIMITE_POR_JANELA = 20;

function ipDoCliente(request: Request): string {
  const encaminhado = request.headers.get("x-forwarded-for");
  if (encaminhado) return encaminhado.split(",")[0].trim();
  return "desconhecido";
}

export async function assertDentroDoLimite(request: Request): Promise<NextResponse | null> {
  const redis = getCliente();
  if (!redis) return null;

  const chave = `bff:ratelimit:${ipDoCliente(request)}`;
  try {
    const contagem = await redis.incr(chave);
    if (contagem === 1) {
      await redis.expire(chave, JANELA_SEGUNDOS);
    }
    if (contagem > LIMITE_POR_JANELA) {
      return NextResponse.json(
        { error: "muitas requisições, tente novamente em instantes" },
        { status: 429 }
      );
    }
  } catch {
    // Redis indisponível/timeout: fail-open, não trava mutação legítima
    // por causa de uma camada redundante — mesma escolha já documentada
    // em ratelimit_redis.go do backend ("deixa passar (fail-open) em vez
    // de bloquear todo mundo").
  }

  return null;
}
