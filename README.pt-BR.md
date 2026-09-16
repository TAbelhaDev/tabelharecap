<div align="center">

# TAbelhaRecap

**Um inbox passivo de "novidades"** — outras ferramentas registram um item
via IPC, você só confere o que apareceu.

[English](README.md) · **Português**

[![Go Version](https://img.shields.io/github/go-mod/go-version/TAbelhaDev/tabelharecap?style=flat-square&logo=go&logoColor=white&color=00ADD8)](go.mod)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Powered by tabelhatuiui](https://img.shields.io/badge/theme-tabelhatuiui-d6b4f7?style=flat-square)](https://github.com/TAbelhaDev/tabelhatuiui)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)](LICENSE)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/ianptkcs)

</div>

---

## O que é

Com automações, jobs recorrentes e projetos suficientes rodando em
paralelo, é fácil perder o rastro do que realmente aconteceu. O `tarecap`
não é mais um dashboard que sai puxando dado de todo mundo — é o oposto: um
inbox pequeno e agnóstico de fonte, no qual qualquer coisa pode escrever.

Qualquer ferramenta, script, job de cron ou passo de workflow pode
registrar uma "novidade" (algo que vale a pena saber) chamando `tarecap ipc
item.add`. O próprio `tarecap` nunca sai buscando nada — ele só guarda o que
lhe é contado e mostra de volta num feed cronológico único, então abrir ele
é um "o que rolou desde a última vez que eu chequei" rápido, não uma caça
ao tesouro por cinco outros TUIs.

O tema e o chrome compartilhado (header/footer/painéis, registro de teclas,
os helpers de `ipc ... --json`) vêm de
[`tabelhatuiui`](https://github.com/TAbelhaDev/tabelhatuiui) /
[`tabelhascaff`](https://github.com/TAbelhaDev/tabelhascaff), compartilhados
entre os meus TUIs em Bubble Tea.

## Conteúdo

- [Instalação](#instalação)
- [Uso](#uso)
- [IPC](#ipc)
- [Configuração](#configuração)
- [Licença](#licença)

## Instalação

Requer Go 1.26+.

```bash
go install github.com/TAbelhaDev/tabelharecap@latest
```

Isso instala o binário como `tabelharecap` (igual ao nome do módulo). Pra
ter o nome curto `tarecap` usado neste README, compile a partir do código:

```bash
git clone https://github.com/TAbelhaDev/tabelharecap.git
cd tabelharecap
go build -o tarecap .
```

## Uso

Rodar `tarecap` sem argumentos abre o TUI: um feed único, mais recente
primeiro, com um marcador `●`/`○` pra não-visto/visto. Teclas:

| Tecla | Ação |
| --- | --- |
| `enter` / `s` | marca o item selecionado como visto |
| `A` | marca todos os itens como vistos |
| `u` | alterna "só não vistos" |
| `f` | percorre as fontes presentes no momento |
| `r` | recarrega do banco (pega qualquer coisa que outro processo acabou de adicionar) |
| `?` | ajuda |
| `,` | reconfigurar teclas |
| `q` | sair |

Fora isso é só leitura: criar e editar item é papel do endpoint de escrita
do IPC abaixo, não do TUI.

## IPC

Qualquer chamador externo — um script de job em bash, um passo de workflow,
um comando avulso — registra uma novidade do mesmo jeito que todo outro TUI
do `ianptkcs` expõe uma fonte de dados scriptável:

```bash
tarecap ipc item.add source=post-suggestions title="3 sugestões novas revisadas" \
  body="opcional, texto mais longo" link="opcional, comando ou path pra abrir a origem" --json

tarecap ipc item.list --json                # todo item, mais recente primeiro
tarecap ipc item.list unseen=true --json    # só os não vistos
tarecap ipc item.list source=tajobs --json  # só os de uma fonte

tarecap ipc item.seen id=42 --json          # marca um item como visto (idempotente)
tarecap ipc item.seen-all --json            # marca todo item não visto como visto
```

`source` e `title` são os únicos campos obrigatórios em `item.add`. `link`
é guardado como uma string opaca — o `tarecap` nunca interpreta nem executa
isso, só exibe de volta; o que significa (um comando de shell, um caminho
de arquivo, uma URL) é decisão de quem for ler depois.

Ligar `taglue`/`tajobs`/`taradar`/`tabelhakanban` (ou notas soltas em
markdown/notificações) pra de fato chamar `item.add` é um passo futuro
deliberado, não algo que esta ferramenta faz sozinha — o `tarecap` não sabe
nada sobre, e nunca fala com, o IPC de nenhuma outra ferramenta.

## Configuração

`~/.config/tabelharecap/config.toml`:

```toml
[database]
path = "~/.local/state/tabelharecap/tarecap.db"
```

`TARECAP_DB` sobrescreve o caminho do banco por completo (inclusive sobre o
arquivo de config) — útil pra apontar uma rodada de teste pra um arquivo
descartável. O banco é SQLite em modo WAL, já que é escrito por vários
processos curtos e independentes (jobs de cron, passos de workflow), não
por um único processo de vida longa.

Atalhos de teclado ficam em `~/.config/tabelharecap/keybindings.json`,
editados pelo modal de configurações do próprio app (`,`) em vez de na mão.

## Licença

[GNU AGPL-3.0](LICENSE) — livre e de código aberto. Se você rodar uma versão
modificada deste projeto, inclusive como serviço de rede, também precisa
disponibilizar o código-fonte modificado sob a mesma licença.
