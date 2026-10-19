#!/usr/bin/env bash
#
# Levanta el servidor de tasks-ai en segundo plano: compila el binario, arranca
# `serve` y deja su registro y su pid en .run/.
#
#   ./serve.sh          arranca el servidor si no está ya en marcha
#   ./serve.sh status   dice si está en marcha y dónde escucha
#   ./serve.sh stop     lo para
#
# La dirección y el puerto son los del settings del proyecto (.task-ai/settings.json).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
binary="$root/bin/tasks-ai"
state="$root/.run"
pidfile="$state/serve.pid"
logfile="$state/serve.log"

cd "$root"

# running dice si el proceso del pid guardado sigue vivo.
running() {
	[[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

# address devuelve la dirección que el servidor anunció al arrancar.
address() {
	grep -o -m1 'http://[^"]*' "$logfile" 2>/dev/null || echo "(todavía sin dirección)"
}

# failed cuenta por qué no arrancó y retira el pid que ya no vale.
failed() {
	echo "El servidor no arrancó. Registro:" >&2
	cat "$logfile" >&2
	rm -f "$pidfile"
}

start() {
	if running; then
		echo "El servidor ya está en marcha (pid $(cat "$pidfile")): $(address)"
		return
	fi

	mkdir -p "$state"
	go build -o "$binary" ./cmd/tasks-ai
	: >"$logfile"

	nohup "$binary" serve >>"$logfile" 2>&1 &
	echo $! >"$pidfile"

	# Se espera al anuncio de la dirección y a que el proceso siga vivo después:
	# el aviso sale antes de atender, así que un puerto ocupado solo se descubre
	# un momento más tarde.
	for _ in {1..50}; do
		running || { failed; return 1; }
		if grep -q 'listening on' "$logfile"; then
			sleep 0.2
			break
		fi
		sleep 0.1
	done

	running || { failed; return 1; }

	echo "Servidor en marcha (pid $(cat "$pidfile")): $(address)"
	echo "Registro: $logfile"
}

stop() {
	if ! running; then
		rm -f "$pidfile"
		echo "El servidor no está en marcha."
		return
	fi

	local pid
	pid="$(cat "$pidfile")"
	kill "$pid" 2>/dev/null || true

	# El servidor atiende la señal y cierra su canal de avisos: se le da tiempo
	# antes de darlo por colgado.
	for _ in {1..50}; do
		kill -0 "$pid" 2>/dev/null || break
		sleep 0.1
	done
	kill -9 "$pid" 2>/dev/null || true

	rm -f "$pidfile"
	echo "Servidor parado."
}

status() {
	if running; then
		echo "En marcha (pid $(cat "$pidfile")): $(address)"
	else
		echo "Parado."
	fi
}

case "${1:-start}" in
start) start ;;
stop) stop ;;
status) status ;;
*)
	echo "Uso: $0 [start|stop|status]" >&2
	exit 2
	;;
esac
