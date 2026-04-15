# Parse API Messages — build helper.
# Требуется: Go 1.22+, Node 18+, Wails CLI v2.12+.
#
# Полезные команды:
#   make dev             — wails dev (hot reload)
#   make deps            — поставить зависимости Go и frontend
#   make doctor          — wails doctor
#   make mac             — нативная сборка под текущую архитектуру macOS
#   make mac-universal   — macOS universal binary (amd64 + arm64)
#   make windows         — кросс-сборка под Windows amd64
#   make all             — mac-universal + windows
#   make zip-mac         — собрать mac-universal и упаковать в zip
#   make zip-windows     — собрать windows и упаковать в zip
#   make release         — all + оба zip-архива (очищает build/release перед сборкой)
#   make clean           — очистить build/bin и build/release

APP_NAME    := parse-api-messages
# Версия берётся из wails.json -> Info.productVersion, по умолчанию dev.
VERSION     := $(shell grep -o '"productVersion":[^,]*' wails.json | sed -E 's/.*"([^"]+)"$$/\1/' 2>/dev/null || echo dev)
RELEASE_DIR := build/release
WAILS       := wails

.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Current version from wails.json: $(VERSION)"

# --------------------------------------------------------------------
# Разработка
# --------------------------------------------------------------------

.PHONY: dev
dev: ## Запустить wails dev с hot reload
	$(WAILS) dev

.PHONY: deps
deps: ## Установить Go и frontend зависимости
	go mod download
	cd frontend && npm install

.PHONY: doctor
doctor: ## wails doctor — проверка окружения
	$(WAILS) doctor

.PHONY: vet
vet: ## go vet ./...
	go vet ./...

.PHONY: fmt
fmt: ## go fmt ./...
	go fmt ./...

# --------------------------------------------------------------------
# Сборка
# --------------------------------------------------------------------

.PHONY: mac
mac: ## Сборка под текущую архитектуру macOS
	$(WAILS) build -clean

.PHONY: mac-universal
mac-universal: ## macOS universal binary (amd64 + arm64)
	$(WAILS) build -clean -platform darwin/universal

.PHONY: windows
windows: ## Кросс-сборка под Windows amd64
	$(WAILS) build -clean -platform windows/amd64

.PHONY: all
all: mac-universal windows ## Собрать mac-universal + windows

# --------------------------------------------------------------------
# Релиз — zip-архивы в build/release/
# --------------------------------------------------------------------

MAC_ZIP     := $(RELEASE_DIR)/$(APP_NAME)-$(VERSION)-darwin-universal.zip
WIN_ZIP     := $(RELEASE_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.zip

.PHONY: zip-mac
zip-mac: mac-universal ## Собрать mac-universal и упаковать в zip
	@mkdir -p $(RELEASE_DIR)
	@rm -f "$(MAC_ZIP)"
	@cd build/bin && zip -rq "../../$(MAC_ZIP)" "$(APP_NAME).app"
	@echo ""
	@echo "→ $$(pwd)/$(MAC_ZIP)"
	@ls -lh "$(MAC_ZIP)"

.PHONY: zip-windows
zip-windows: windows ## Собрать windows и упаковать в zip
	@mkdir -p $(RELEASE_DIR)
	@rm -f "$(WIN_ZIP)"
	@cd build/bin && zip -q "../../$(WIN_ZIP)" "$(APP_NAME).exe"
	@echo ""
	@echo "→ $$(pwd)/$(WIN_ZIP)"
	@ls -lh "$(WIN_ZIP)"

.PHONY: release
release: clean-release zip-mac zip-windows ## Собрать всё и упаковать оба zip (build/release/)
	@echo ""
	@echo "Готово. Архивы:"
	@ls -lh $(RELEASE_DIR)
	@echo ""
	@echo "Отдать можно напрямую:"
	@echo "  $$(pwd)/$(MAC_ZIP)"
	@echo "  $$(pwd)/$(WIN_ZIP)"

.PHONY: clean-release
clean-release:
	@rm -rf $(RELEASE_DIR)

# --------------------------------------------------------------------
# Очистка
# --------------------------------------------------------------------

.PHONY: clean
clean: clean-release ## Удалить build/bin и build/release
	@rm -rf build/bin
