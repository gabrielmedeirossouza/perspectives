# Julgamento da implementação

**Estado após correção:** os seis achados abaixo foram corrigidos e cobertos por regressões. A validação atual está em **Evidência e limites**; os achados preservam o diagnóstico original.

Avaliação original e correções em 2026-11-01 sobre as alterações locais. Referência: plano `dependencias-por-regra.md`. Os caminhos abaixo partem de `.sdd/templates/.agents/tools/`.

**Escopo corrigido pelo usuário:** ainda não há consumidores do framework; não implementar retrocompatibilidade nem migração legada. Os requisitos correspondentes do plano deixam de ser critérios de aprovação.

## Achados originais

### 1. [P1] Gravações concorrentes perdem vereditos silenciosamente

**Local:** `internal/meta/review.go:167–200`.

A comparação de `RawSum` ocorre antes da escrita/renomeação, sem exclusão mútua. Dois escritores podem validar a mesma revisão e sobrescrever um ao outro, ambos retornando sucesso. Em três execuções com 32 escritores, **8, 7 e 8 atualizações** foram aceitas a partir da mesma versão; deveria ser uma. A remoção tem a mesma janela entre conferir e apagar.

**Correção:** serializar por checkpoint a conferência e a alteração, incluindo fechamento, com proteção entre processos. Acrescentar teste concorrente real; renomeação atômica isolada não evita perda de atualização.

### 2. [P1] Mudanças nas fontes de suporte não invalidam o julgamento

**Local:** `internal/graph/review.go:447–516`.

`candidateSupportMap` registra chrome e títulos da cobertura, sem hashes das specs técnicas usadas no julgamento. `candidateLinksHash` cobre apenas o vínculo com a fonte revisada; ignora os demais vínculos e `files`. Reproduzido: julgar código numa revisão de perspectiva, alterar a regra da spec técnica de suporte e atualizar seu metadado. O resultado continua **1/1 resolvido**, e `CloseReview` aceita evidência do contexto anterior.

**Correção:** persistir e validar as fontes e os vínculos efetivamente usados, incluindo atribuições por arquivo em nós de diretório. Invalidar somente os vereditos dependentes.

### 3. [P1] Fechamento aceita candidatos com drift bloqueante

**Local:** `internal/graph/review.go:1076–1108`.

`CloseReview` verifica drift apenas na fonte, não nos candidatos. Reproduzido: após um `keep`, inserir no candidato uma fonte duplicada com título desconhecido. O grafo denuncia ambos os erros, mas a revisão permanece **1/1 resolvida**, fecha e remove o checkpoint.

**Correção:** revalidar a estrutura e os efeitos aplicados em cada candidato antes de fechar. Pendências legítimas de implementação podem continuar abertas; metadados inválidos não podem contar como correção validada.

### 4. [P1] Checkpoint corrompido permite encerrar a revisão como preservada

**Local:** `internal/graph/review.go:125–131,618–627`.

Um checkpoint ilegível gera `reviewDrift`, mas deixa `spec.review == nil`. Abrir revisão expandida, corromper seu JSON e executar `--scope preserved` retorna sucesso e avança `reviewed`, apesar dos candidatos sem julgamento. O arquivo inválido ainda é denunciado; apagá-lo depois elimina a evidência da revisão pendente.

**Correção:** distinguir checkpoint ausente de checkpoint inválido e bloquear o avanço de `reviewed` até reconstruir a evidência perdida. O plano proíbe tratar corrupção como conclusão.

### 5. [P2] Conversão integral → seletivo abre revisão indevida

**Local:** `internal/graph/graph.go:876–879`; `internal/graph/ops.go:68–73`.

Adicionar `uses` a um vínculo integral já sincronizado conserva seu snapshot. `rebuildIndex` interpreta qualquer snapshot como sincronização seletiva anterior, mesmo com `hash` integral. Por isso, `ensureReviewed` não inicializa a fonte. Reproduzido com `--approve`: fonte intacta termina com **`reviewed=""` e `OwesReview=true`**. Isso ocorre também em projetos criados com a implementação atual, sem upgrade.

**Correção:** distinguir a conversão inicial de um vínculo seletivo já sincronizado e inicializar a base antes de perder essa distinção. Cobrir `--approve` e `--sync` preservando snapshots existentes.

### 6. [P2] Revisão técnica com escopo desconhecido inclui a própria fonte

**Local:** `internal/graph/review.go:318–330`.

`ClassifyReview` exclui a fonte dos candidatos, mas `ReviewState` volta a incluí-la no ramo `unknown`, pois técnica pode consumir técnica. Reproduzido ao abrir revisão desconhecida: aparece um candidato adicional correspondente à própria spec, impedindo o fechamento após julgar somente consumidores válidos.

**Correção:** aplicar `n != spec` também na descoberta incremental e recusar autovínculos em `RecordVerdict`.

## Evidência e limites

- `go run -C .agents/tools ./guard --check`: passou com vet e testes dos dois módulos; versão 58, sem alteração manual.
- Regressões: seis achados, invalidação somente dos candidatos dependentes, atribuições por arquivo, retomada de aplicação interrompida e fechamento concorrente sem avanço de `reviewed`.
- Concorrência: 24 processos partindo do mesmo checkpoint, incluindo disputa entre gravação e fechamento; exatamente um sucesso. Três repetições com `-race` passaram.
- Escala: regra técnica, perspectiva global e perspectiva de feature, cada uma com 5.000 consumidores. Exatamente **50 nós de código pendentes**; nas perspectivas, apenas a fonte técnica pertinente também ficou pendente. Todos os **4.950 metadados restantes** foram comparados integralmente após classificação e processamento dos arquivos staged.
- Benchmark, três iterações por modo, Apple M4: integral **311,6 ms / 77,65 MB / 557.839 alocações**; seletivo **300,7 ms / 73,05 MB / 512.113 alocações**. O aceite é a redução do trabalho encaminhado; esta amostra não sustenta promessa de velocidade.
- Projeto temporário `/tmp/sdd-review-validation.aGYV8x`: procedimentos de spec → design → implementação → auditoria com juiz independente → commit; TypeScript compilado, testes Node verdes, revisão **3/3**, duas aprovações, checkpoint removido e `No pending work`. Hook e push para remoto local descartável passaram; último commit `5d4676b`. O sandbox `finance`, que contém alterações locais, permaneceu intacto.
- Checkpoints usam exclusão mútua por diretório de lock. Se o processo for terminado durante a escrita, a recuperação do lock segue o catálogo de ferramentas; não há conclusão automática nem migração legada.

Os testes novos estão em `internal/meta/review_test.go`, `internal/graph/review_test.go` e `internal/graph/scale_test.go`; o cenário técnico existente verifica agora todos os metadados não afetados.
