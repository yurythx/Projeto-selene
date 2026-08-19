import { describe, it, expect, vi, afterEach } from "vitest";

// ioredis mockado — sem isso o teste tentaria uma conexão TCP de verdade
// (mesmo com lazyConnect, o primeiro comando dispara a conexão) contra um
// Redis que não existe no ambiente de teste unitário, travando/falhando
// por timeout em vez de exercitar a lógica de contagem/limite.
const incrMock = vi.fn();
const expireMock = vi.fn();

vi.mock("ioredis", () => ({
  // função regular, não arrow — o código real chama `new Redis(...)`, e
  // uma arrow function não é "constructible" (TypeError: is not a
  // constructor) mesmo dentro de mockImplementation.
  default: vi.fn().mockImplementation(function RedisMock() {
    return { incr: incrMock, expire: expireMock, on: vi.fn() };
  }),
}));

function req() {
  return new Request("http://localhost:3000/api/contratos", {
    method: "POST",
    headers: { "x-forwarded-for": "203.0.113.9" },
  });
}

// vi.resetModules() + import dinâmico em cada teste: o módulo memoiza o
// cliente Redis num `let` de nível de módulo (getCliente) — sem isso, o
// primeiro teste que rodasse decidiria o valor de REDIS_ADDR pra todos os
// outros, já que o singleton sobreviveria entre os `it()`.
describe("assertDentroDoLimite", () => {
  const redisAddrOriginal = process.env.REDIS_ADDR;

  afterEach(() => {
    vi.resetModules();
    incrMock.mockReset();
    expireMock.mockReset();
    if (redisAddrOriginal === undefined) delete process.env.REDIS_ADDR;
    else process.env.REDIS_ADDR = redisAddrOriginal;
  });

  it("sem REDIS_ADDR configurado, deixa passar (fail-open)", async () => {
    delete process.env.REDIS_ADDR;
    const { assertDentroDoLimite } = await import("./rate-limit");
    expect(await assertDentroDoLimite(req())).toBeNull();
    expect(incrMock).not.toHaveBeenCalled();
  });

  it("dentro do limite, deixa passar", async () => {
    process.env.REDIS_ADDR = "localhost:6379";
    incrMock.mockResolvedValue(5);
    const { assertDentroDoLimite } = await import("./rate-limit");
    expect(await assertDentroDoLimite(req())).toBeNull();
  });

  it("primeira requisição da janela seta o TTL da chave", async () => {
    process.env.REDIS_ADDR = "localhost:6379";
    incrMock.mockResolvedValue(1);
    const { assertDentroDoLimite } = await import("./rate-limit");
    await assertDentroDoLimite(req());
    expect(expireMock).toHaveBeenCalledWith(expect.stringContaining("203.0.113.9"), 10);
  });

  it("acima do limite, bloqueia com 429", async () => {
    process.env.REDIS_ADDR = "localhost:6379";
    incrMock.mockResolvedValue(21);
    const { assertDentroDoLimite } = await import("./rate-limit");
    const resposta = await assertDentroDoLimite(req());
    expect(resposta).not.toBeNull();
    expect(resposta!.status).toBe(429);
  });

  it("erro no Redis (indisponível), deixa passar (fail-open)", async () => {
    process.env.REDIS_ADDR = "localhost:6379";
    incrMock.mockRejectedValue(new Error("connection refused"));
    const { assertDentroDoLimite } = await import("./rate-limit");
    expect(await assertDentroDoLimite(req())).toBeNull();
  });
});
