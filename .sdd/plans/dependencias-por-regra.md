# Upgrade: dependências por regra

**Status:** proposta de implementação. Este documento planeja o upgrade; não altera o contrato vigente.

## Objetivo e escopo

Permitir que consumidores dependam de regras específicas de specs técnicas e de perspectivas. No cenário de referência, alterar uma regra com 5.000 consumidores deve gerar revalidação de implementação apenas nos 50 arquivos que a selecionam, desde que o escopo permaneça igual, os vínculos estejam auditados e dependências técnicas não acrescentem bloqueios fora desse conjunto.

**Decisões confirmadas pelo usuário:** seleção para ambas as espécies de spec e persistência do progresso parcial das revisões. Cobrir código → técnica, técnica → técnica, técnica → perspectiva e código → perspectiva. Incluir specs globais na raiz de cada perspectiva (`specs/<perspectiva>/spec.md`) e specs de features; preservar os caminhos configurados. Perspectivas continuam sem declarar fontes.

Separar três custos: descoberta de aplicabilidade, revalidação de implementação e leitura mecânica da base. Reduzir o segundo é o objetivo; otimizar o terceiro depende de medição.

## Diagnóstico atual

| Componente | Limitação |
|---|---|
| `internal/meta/meta.go` | `Source` registra caminho, hash do arquivo e todas as regras; não declara uma seleção nem separa `## Adopted patterns`. |
| `internal/graph/graph.go` | `Edge.Stale` compara hashes do arquivo; `codeUnit`, `wait`, `Design` e os lotes propagam esse resultado; `consumers` e `Units` são recalculados várias vezes por execução. |
| `internal/graph/ops.go` | `Sync` copia o catálogo completo; `audited` só zera quando uma fonte nova é sincronizada. |
| `internal/graph/audit.go` | Dossiês e candidatos tratam vínculos por spec; cada dossiê relê todas as specs técnicas do disco. |
| `metadata`, `pending` e skills | Criam, corrigem e sincronizam vínculos nessa granularidade; `/sdd-implement` lê todos os consumidores da spec alvo. |

Os caminhos acima são relativos a `.sdd/templates/.agents/tools/`. Extração (`md.Rules`), hashes (`meta.RuleHashes`) e deltas (`meta.Diff`) por regra já existem: reaproveitá-los.

Conteúdo fora de `###` não pertence a nenhuma regra: propósito, **Out of scope** (chrome por `specs/AGENTS.md`), introduções de `##` e `## Adopted patterns`. Hoje essas edições tornam todos os consumidores `stale`; no modo seletivo seriam invisíveis sem o tratamento abaixo.

## Contrato proposto

### Seleção explícita

Adicionar `sources[].uses`, autorado pelas skills; `sources[].rules` continua snapshot da ferramenta.

```json
{
  "path": "src/specs/code-guide/spec.md",
  "uses": ["Constructor injection", "Consumer-owned abstraction"]
}
```

| Campo ou situação | Semântica |
|---|---|
| `uses` ausente | Dependência integral; `stale` compara `hash` e exige que `rules` cubra todo o catálogo atual da fonte. |
| `uses` presente | Lista não vazia de títulos exatos da fonte técnica ou de perspectiva, sem duplicatas. Diferença apenas de caixa é drift com o título correto sugerido. |
| `rules` após sync seletivo ou `--approve` | Exatamente os hashes atuais de `uses`; entradas excedentes são podadas no `--sync` e no `--approve`. |
| `hash` da fonte | Gravado apenas no sync integral; omitido no modo seletivo (`--sync` seletivo e `--approve` limpam `hash`). Sua ausência num vínculo sem `uses` identifica fonte nova ou transição seletivo → integral, tornando o vínculo `stale` e zerando `audited` no próximo sync. |
| Título em `uses` sem snapshot | Regra nova para o consumidor: trabalho de design se ele é técnico, de implementação se é código. |
| Título em `uses` ausente da fonte | Regra removida ou renomeada: trabalho do escritor do consumidor; `--sync` recusa até `uses` ser corrigido. |
| Snapshot fora de `uses`, regra ainda na fonte | Seleção reduzida por veredito: ignorado, sem trabalho; podado em `--approve` ou no próximo `--sync`. |
| Snapshot fora de `uses`, regra ausente da fonte | Remoção ou renome pendente após edição de `uses` pelo escritor do consumidor: limpo no `--sync`. |

A identidade continua sendo caminho da spec + título; `uses` é escopado por fonte, logo títulos iguais em specs diferentes não colidem. Para detectar renomes, comparar o título selecionado sumido com o catálogo atual completo, não só com `uses`; corpo igual sugere a correspondência, nunca a aplica. No `--sync`, se um título sem snapshot em `uses` tem o mesmo hash de corpo de um título removido do snapshot anterior, trata-se de renome confirmado e `audited` é preservado.

### Cobertura de perspectivas

Estender `served`, `missing` e `unserved` para conjuntos de regras por perspectiva. Cada nó de código deve listar diretamente as regras de produto atendidas por suas fontes técnicas; a ferramenta não substitui esses vínculos por dependências transitivas.

- A cobertura exigida é a união das seleções técnica → perspectiva das specs técnicas diretamente listadas pelo código. Se alguma for integral, a cobertura exigida dessa perspectiva também é integral.
- Para cobertura seletiva, detectar títulos exigidos ausentes (`missing`) mesmo quando o caminho da perspectiva já está listado. Se as fontes técnicas do nó atendem apenas um subconjunto da perspectiva, listar a perspectiva de forma integral no código ou incluir títulos fora dessa união acusa regras sem suporte técnico (`unserved`), exigindo correção de design ou restrição de `uses`.
- Selecionar regras de uma spec técnica não reduz automaticamente quais regras de produto ela atende. Essa correspondência interna entre regras permanece fora deste upgrade; a união por spec é conservadora.
- Testes herdam os vínculos e seleções do código testado, além das regras aplicáveis do executor de testes.

### Invariantes

- `uses` declara dependência, não delimita obrigação. C1/C3 continuam exigindo toda regra aplicável; regra aplicável fora de `uses` é defeito de seleção corrigido pela auditoria, nunca permissão para violá-la.
- Na dúvida, selecionar ou manter integral (respeitando o suporte técnico nas perspectivas): seleção a mais custa revalidação; seleção a menos perde obrigações em silêncio.
- Alterações fora das regras selecionadas não tornam o consumidor `stale` nem regravam seu metadado.
- `--write` e o hook nunca validam implementação nem alteram `uses` ou snapshots; `--sync` continua exclusivo dos escritores autorizados e nunca amplia `uses`.
- `audited` zera no sync que registra fonte nova, título novo em `uses` (exceto renome de mesmo corpo) ou transição seletivo → integral (`uses` ausente com `hash` vazio). Reduções por veredito não zeram e `--approve` poda `sources[].rules` e limpa `hash` seletivo. Sem campo novo de hash de auditoria.
- Drift estrutural da fonte continua bloqueante, inclusive em regras não selecionadas.
- A ferramenta não deduz dependências de links. Um link com `#âncora` de regra selecionada para regra não selecionada aparece como dica no dossiê da auditoria.
- `## Adopted patterns` obriga autores de specs técnicas: `md` extrai o corpo dessa seção `##` e o metadado da spec técnica (e `sources[]` de consumidores técnicos) grava seu hash em campo opcional `adopted`, separado de `rules`. Em vínculo técnica → técnica seletivo, mudança em `adopted` torna o consumidor `stale` com delta próprio, sem gerar regra `unconsumed` para código.

### Nós de diretório

Manter `files` como hoje (arquivo → specs). A seleção fica em `sources[].uses` do nó, cobrindo a união das regras de seus arquivos. No `/sdd-audit`, o juiz declara as regras aplicáveis por arquivo vinculado e a correção consolida a união em `sources[].uses` do diretório; o diretório só é aprovado quando as regras adicionadas já constam sincronizadas no nó. Arquivos TypeScript continuam com nós próprios.

### Estados esperados

| Evento | Resultado |
|---|---|
| Corpo de regra selecionada muda | Consumidor `stale`, com o delta dessa regra. |
| Só regra não selecionada muda | Consumidor intacto; revisão de aplicabilidade devida (abaixo). |
| Edição fora de `###` | Modo integral: `stale`, como hoje. Modo seletivo: só revisão de aplicabilidade (exceto `## Adopted patterns` em técnica → técnica). |
| Regra selecionada removida ou renomeada | Trabalho de design ou implementação conforme o consumidor; renome nunca resolvido automaticamente. |
| Arquivo da fonte desaparece | `dangling`, como hoje. |
| Regra de spec técnica de feature sem consumidor de código que a selecione ou liste integralmente | `unconsumed` por regra, para `/sdd-implement` (acompanhada apenas das regras de perspectiva servidas pela spec ainda sem consumidor de código). Specs técnicas globais continuam `unconsumed` apenas quando nenhum nó de código lista a spec; regras globais individuais são cobradas por auditoria e revisão de aplicabilidade. |
| Regra de perspectiva sem consumidor técnico que a selecione ou liste integralmente | `unconsumed` por regra, para `/sdd-design`; vínculo direto de código não substitui design. |
| Código lista a perspectiva, mas omite regra exigida por suas fontes técnicas | Pendência de cobertura por título (`missing`), mesmo sem faltar o caminho da fonte. |
| Spec técnica intermediária seleciona a regra alterada | Ela fica `stale` e seus consumidores esperam `/sdd-design`, como hoje. |

Isolar consumidores de uma spec técnica pendente exigiria mapear suas regras às de origem; fora deste upgrade.

## Descoberta de aplicabilidade

A seleção resolve impacto conhecido; não descobre quem passou a precisar de uma regra. O upgrade só está completo com este fluxo.

**Detecção.** O metadado de cada spec técnica ou de perspectiva ganha `reviewed`: o hash da spec cuja aplicabilidade foi revisada. Enquanto a spec não possui consumidores seletivos nem checkpoint aberto, `Refresh` mantém `reviewed` igual ao `hash` atual (e o primeiro `--sync` ou `--approve` seletivo contra uma spec íntegra sem `reviewed` inicializa `reviewed` com o hash atual da spec), evitando que a adoção gradual dispare revisão retroativa da base. Quando a spec já possui consumidor seletivo, `Refresh` preserva `reviewed`; hash atual ≠ `reviewed` indica revisão devida. Um checkpoint aberto continua visível até o fechamento, mesmo se seus vínculos seletivos forem removidos durante a revisão.

**Fluxo.**

1. O escritor da fonte classifica a diferença desde `reviewed` (`/sdd-spec` para perspectivas, `/sdd-design` para técnicas), obtida pela ferramenta localizando no histórico Git o conteúdo da spec cujo hash de nó é `reviewed`. Sem versão anterior no Git, o escopo é desconhecido. Escopo preservado sem checkpoint aberto permite fechar a revisão gravando `reviewed`; se já houver checkpoint aberto ou escopo ampliado/desconhecido, abrir/atualizar os candidatos em caminhos, preservando pendências anteriores. Sem delimitação confiável, incluir todos os consumidores potenciais permitidos, técnicos e de código, mesmo sem vínculo atual.
2. Manter no máximo um checkpoint aberto por spec fonte em `.sdd/reviews/<id>.json` (identidade determinística derivada do caminho da spec), operado exclusivamente pelo `metadata`. Ele registra fonte, hash julgado, regras revisadas e escopo candidato. A spec mantém apenas `reviewed`; não armazena seus consumidores.
3. `/sdd-design` julga candidatos técnicos e corrige seus vínculos. `/sdd-audit` julga candidatos de código, incluindo regras de perspectivas. Uma regra de produto aplicável sem suporte técnico é encaminhada para design; não se cria um vínculo de código sem fonte técnica que a atenda.
4. Salvar cada veredito antes das correções e registrar os efeitos aplicados após validá-los. Persistir adições, reduções, exclusões justificadas e bloqueios. Retomar apenas candidatos sem resultado válido ou com correções pendentes, sem refazer julgamentos válidos.
5. Quando todos os candidatos estão resolvidos e validados, o fechamento no `metadata` (invocado pelo escritor da fonte ou ao concluir o último candidato sem bloqueios) confere o checkpoint, grava `reviewed` no metadado da spec e remove `.sdd/reviews/<id>.json`. Isso não sincroniza consumidores nem aprova implementação; trabalho gerado pelas adições segue seus gates.

**Checkpoint e retomada.** Um arquivo por revisão aberta evita concentrar histórico num registro global. A fonte e as regras são registradas uma vez; cada candidato contém seu caminho/tipo, hashes do conteúdo, dos vínculos e das fontes usadas no julgamento, veredito e evidência das correções aplicadas.

- Na retomada, recalcular candidatos e validar os hashes. Alterações no arquivo, na seleção ou nas fontes usadas no julgamento invalidam os resultados dependentes; arquivos novos entram na fila. Remoções exigem verificar a limpeza dos vínculos.
- Mudança no hash da fonte reabre a classificação sobre o mesmo checkpoint; vereditos só são reutilizados quando as regras julgadas e o contexto de aplicabilidade continuam comprovadamente iguais. Sem essa prova, rejulgar os candidatos afetados.
- Correções precisam ser idempotentes. Interrupção entre veredito, edição do metadado e checkpoint deve permitir detectar o que já foi aplicado, sem repetir adições ou aprovações.
- Gravar checkpoints atomicamente, detectar atualizações concorrentes e recusar fechamento com fonte desatualizada, conflitos, candidatos não julgados, correções pendentes ou bloqueios sem resolução.
- `graph.Load` e `g.Staged()` carregam e validam `.sdd/reviews/`, reportando como `drift` checkpoints corrompidos, inválidos ou órfãos. `pending` mostra progresso e responsável sem alterar o checkpoint. Sua perda ou corrupção exige refazer a parte sem evidência; nunca significa revisão concluída. `audited = 0` continua encaminhando código novo ou alterado para auditoria comum.

**Base inicial.** Spec sem `reviewed` e sem consumidores seletivos permanece válida sem regravação. Ao ganhar seu primeiro consumidor seletivo com a spec íntegra, `reviewed` assume o hash atual da spec; se a spec tiver sido alterada sem revisão após ter consumidor seletivo, a revisão é devida com escopo desconhecido até julgamento completo.

## Etapas de implementação

### 1. Fechar e documentar o contrato

- **Metadata** e **Working files** no `AGENTS.md` do template: `uses` para ambas as fontes, `adopted`, `reviewed`, ciclo de vida de `.sdd/reviews/<id>.json` (remoção no fechamento), estados, escritores, poda no `--approve` e regra de zeragem de `audited`.
- C1, C3, C6 e Implementation gate em `src/AGENTS.md`: obrigação por aplicabilidade, seleção como declaração auditada, sync só com todas as regras selecionadas implementadas.
- `specs/AGENTS.md`: escopo normativo dentro das regras; renomes chegam ao escritor do consumidor; classificação de aplicabilidade por `/sdd-spec` sem permissão para editar metadados de código.
- Fechar o formato e o ciclo de vida do checkpoint: identidade única por spec, resolução da versão `reviewed` via Git, validade dos vereditos, gravação atômica, concorrência, fechamento com remoção do arquivo e recuperação após interrupção. O catálogo deve atribuir as operações à ferramenta, não a edições manuais das skills.
- Flags e saídas novas só no catálogo das ferramentas.

**Saída:** exemplos válidos e inválidos e tabela de transições, sem decisões de formato em aberto.

### 2. Evoluir representação e cálculo de impacto

- `internal/md` e `internal/meta`: extrair `## Adopted patterns`; ler e serializar `uses`, `adopted` e `reviewed`; `Diff` filtrado pela seleção, com detecção de renome contra o catálogo completo e delta de `adopted`. Modelar checkpoints em formato próprio e validá-los no carregamento do grafo.
- `internal/graph/graph.go`: `Edge.Stale` e `Delta` cientes da seleção nos quatro tipos de vínculo (com `hash` ausente marcando seletivo); `codeUnit`, `wait`, `Design`, lotes e `unconsumed` por regra (restrito a specs de feature para regras individuais, mantendo specs globais por nó); calcular `Units` uma única vez por execução e indexar consumidores por fonte/regra.
- Evoluir `served`, `missing` e `unserved` conforme **Cobertura de perspectivas**, incluindo união de títulos, predominância do vínculo integral na fonte técnica e bloqueio de cobertura integral no código quando a fonte técnica é seletiva. Preservar a proibição de fontes em nós de perspectiva.
- Preservar ciclos, bloqueios de design e estacionamento de gaps. Regras não selecionadas não entram em unidades; um Design gap só casa com regras de deltas seletivos ou de `unconsumed` por regra.

**Saída:** o grafo identifica exatamente os consumidores e deltas relevantes, com o modo integral inalterado.

### 3. Atualizar operações e relatórios

- `ops.go`: `Refresh` preserva `uses`, mantém `reviewed` sincronizado enquanto não há consumidor seletivo e preserva `reviewed` quando há; `Sync` grava apenas snapshots selecionados (omitindo `hash` no modo seletivo), recusa títulos inválidos e aplica a zeragem de `audited` (preservando em renome de mesmo corpo); `Approve` poda `sources[].rules` para `uses` e limpa `hash` seletivo.
- `metadata/main.go`: validar seleções, resolver diff desde `reviewed` via Git e oferecer operações explícitas de abrir, registrar progresso e fechar revisões (removendo o checkpoint ao fechar). Implementar retomada idempotente e validação de checkpoints; recusar fechamento com drift na spec ou evidência incompleta.
- `pending/main.go`: regras afetadas, regras sem consumidor, revisões devidas com dono da fonte, progresso salvo e próximo candidato por escritor de consumidor, sem recalcular `Units`.
- `audit.go`: dossiê com regras técnicas e de produto selecionadas, não selecionadas como candidatas e links `#âncora` entre regras; modo focado em revisão, reaproveitando resultados válidos; reutilizar texto/propósito das specs já lido em `graph.Load`.
- `--staged` sem sync ou fechamento implícitos e sem tocar nos 4.950 consumidores inalterados. Validar checkpoints em `.sdd/reviews/` sem exigir que uma revisão esteja concluída para permitir commit.

**Saída:** CLI e hook preservam as garantias do grafo e informam a próxima ação de cada dono.

### 4. Integrar skills

| Skill | Ajuste |
|---|---|
| `/sdd-spec` | Classificar alterações de perspectivas globais e de features; abrir ou fechar suas revisões pela ferramenta e encaminhar candidatos aos escritores, sem editar código ou metadados de consumidores. |
| `/sdd-design` | Classificar alterações técnicas; selecionar regras em vínculos técnica → técnica e técnica → perspectiva; julgar candidatos técnicos e salvar progresso. |
| `/sdd-implement` | Ler apenas consumidores pendentes na unidade/regra alvo; selecionar regras ao criar ou alterar código; executar deltas e regras sem consumidor; corrigir `uses` em renomes e remoções; sincronizar após o gate. |
| `/sdd-audit` | Vereditos por regra técnica ou de produto (por arquivo vinculado em nós de diretório, consolidando a união no nó); certificar completude de `uses`; salvar resultados e retomar candidatos de código; encaminhar falta de suporte técnico para design. |
| `/sdd-update-framework` | Instalar ferramentas e leis novas sem migrar metadados existentes; preservar checkpoints válidos e revisões abertas. |
| `/sdd-commit` | Manter `uses` e snapshots junto de seus consumidores; metadados de spec com a spec; incluir criação/atualização/remoção de checkpoints em `.sdd/reviews/` sem perder a consistência entre vereditos e correções. |

Ferramentas cuidam do determinístico; juízes, da aplicabilidade. Cada skill respeita a propriedade dos artefatos e usa a ferramenta para persistir revisões. Nenhuma regra de TypeScript, EDA ou DI nas skills.

**Saída:** um ciclo completo de alteração, revisão, implementação e auditoria funciona apenas pelas skills.

### 5. Compatibilidade

- `uses`, `adopted` e `reviewed` são campos opcionais: manter `version: 2`. Metadados existentes são integrais e continuam válidos sem regravação; nenhum `--migrate`.
- Checkpoints têm versão própria e existem em `.sdd/reviews/` somente enquanto uma revisão está aberta. Atualizar o framework não cria revisões artificiais nem descarta progresso válido.
- Ferramentas antigas recusam metadados com `uses`, `adopted` ou `reviewed` (`DisallowUnknownFields`); o update instala as novas antes de qualquer escrita.
- A granularidade muda gradualmente, por auditoria; nunca copiar o catálogo para `uses`.
- Mudança aditiva: sem `!`, salvo se a etapa 1 decidir quebrar formato. O inteiro de versão do framework continua com o hook.

**Saída:** um projeto atualizado mantém todas as obrigações e nenhum metadado regravado.

### 6. Validar correção, escala e entrega

Testes comportamentais nas suítes de `internal/meta`, `internal/graph`, `metadata` e `pending`, com diretórios temporários gerados em Go. O cenário de 5.000 arquivos fica em benchmark ou atrás de `testing.Short`.

| Caso | Critério de aceite |
|---|---|
| 5.000 arquivos; 50 selecionam a regra técnica alterada; escopo preservado | Exatamente 50 nós com trabalho; 4.950 metadados intactos; revisão fechável por `/sdd-design`. |
| Mesmo cenário com spec global de perspectiva ou de feature | Seleções de produto funcionam; só os 50 consumidores e suas fontes técnicas pertinentes ficam pendentes; demais metadados intactos; classificação por `/sdd-spec`. |
| Perspectiva já listada, título exigido ausente ou código integral com suporte técnico seletivo | `missing` aponta a regra ausente; `unserved` aponta regras sem suporte técnico; sync recusado até corrigir cobertura. |
| Mesma alteração com uma spec técnica intermediária que seleciona a regra | Ela fica `stale`; seus consumidores esperam; os demais não. |
| Regra não selecionada, chrome ou `##` alterados | Nenhum falso `stale` seletivo; revisão devida. |
| `## Adopted patterns` alterado | Consumidores técnicos seletivos ficam `stale` via `adopted`; consumidores de código não são afetados nem surgem falsos `unconsumed`. |
| Regra selecionada técnica ou de produto removida ou renomeada | Pendência para o escritor do consumidor; sync recusado até corrigir `uses`; renome com mesmo corpo preserva `audited` no sync. |
| Regra nova em spec técnica de feature ou de perspectiva | `unconsumed` por regra para o dono correto e revisão devida, incluindo candidatos sem vínculo; regra não selecionada de spec técnica global não gera `unconsumed` falso quando a spec já é consumida. |
| Seleção reduzida por veredito (incluindo migração gradual de 4.950 nós) | Sem trabalho; aprovação mantida; `--approve` poda `sources[].rules` e limpa `hash`, impedindo falso `stale` quando regras não selecionadas forem removidas depois. |
| Seleção ampliada ou seletivo → integral (inclusive sem mudança na fonte) | `hash` ausente sem `uses` marca `stale`; trabalho de implementação; `audited` zera no sync. |
| `uses` inválido, fonte declarada por nó de perspectiva, fonte ausente, ciclo ou drift | Diagnóstico específico; sem sync, aprovação ou fechamento de revisão. `uses` em fonte de perspectiva é válido. |
| Revisão interrompida após julgamentos parciais | Retomar somente candidatos pendentes ou invalidados; preservar adições e exclusões justificadas, inclusive após commit e reinício. |
| Arquivo, seleção ou fonte de suporte muda após checkpoint | Invalidar apenas os vereditos dependentes; candidato novo entra na fila; remoção exige verificar limpeza. |
| Fonte muda ou escopo é ampliado durante revisão (inclusive nova edição com escopo preservado) | Reclassificar sobre o mesmo checkpoint e julgar o conjunto afetado; nunca fechar descartando candidatos pendentes nem usando evidência de outra versão; remover o checkpoint ao fechar. |
| Interrupção entre veredito e correção, escrita concorrente ou checkpoint corrompido | Retomada idempotente, sem aprovações duplicadas; conflito ou falta de evidência nunca fecha revisão. |
| Fontes integral e seletiva no mesmo projeto | Cada modo conserva sua semântica; adoção do primeiro consumidor seletivo em spec íntegra inicializa `reviewed` sem revisão retroativa da base. |
| Projeto v2 existente após update | Pendências idênticas às anteriores; zero arquivos regravados. |
| Hook e update com layouts configurados | Sem sync implícito, caminhos fixos ou perda de adaptações; `--staged` valida `.sdd/reviews/`. |

Medir atual e seletivo no mesmo ambiente: tempo, alocações, arquivos regravados e nós encaminhados ao agente. O aceite funcional é a redução para 50, sem prometer tempo absoluto. Antes de qualquer cache persistente, eliminar trabalho repetido numa execução (índice consumidor por fonte/regra, specs lidas uma vez em `graph.Load`, `Units` calculado uma vez por execução).

Antes da entrega, executar `go vet ./...` e `go test ./...` nas ferramentas, `go run -C .agents/tools ./guard --check` na raiz do framework e o fluxo completo das skills num sandbox, trazendo correções pelo `main`.

**Saída:** testes passando, comparação de escala documentada e entrega pelo fluxo `/sdd-commit`.

## Ordem de entrega

1. Contrato e exemplos.
2. Formato, grafo e testes de impacto seletivo.
3. Operações, revisão de aplicabilidade com checkpoints e auditoria.
4. Skills.
5. Validação de escala.

Entregar tudo na mesma branch; liberar somente com a revisão de aplicabilidade pronta. A otimização não pode trocar revalidação excessiva por obrigações esquecidas.
