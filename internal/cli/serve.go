package cli

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"tasks-ai/internal/http"
	"tasks-ai/internal/store"
	"time"

	"github.com/spf13/cobra"
)

// shutdownWait is how long the server keeps answering to the requests it
// already took before it goes away.
const shutdownWait = 5 * time.Second

// newServeCommand starts the product's web surface.
func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Arranca la superficie web del producto",
		Long: "Sirve la API del producto y su interfaz en la dirección y el puerto del\n" +
			"settings: la interfaz responde en la raíz y la API conserva sus rutas\n" +
			"/api/... Los cambios se anuncian por un canal abierto, de modo que dos\n" +
			"sesiones se ven. Lo que escribe va por la salida de errores, según el nivel\n" +
			"de registro del settings.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return serve(command)
		},
	}
}

// serve opens the project's data saying the server asks for the changes, and
// serves the API until someone stops it. No es una operación: no responde JSON
// por la salida estándar, habla por la salida de errores y según el nivel.
func serve(command *cobra.Command) error {
	dir, err := workingDir()
	if err != nil {
		return err
	}

	opened, err := openProject(command.Context(), dir, command.ErrOrStderr(), store.OriginServer)
	if err != nil {
		return err
	}

	defer func() { _ = opened.Close() }()

	log := slog.New(slog.NewTextHandler(command.ErrOrStderr(), &slog.HandlerOptions{
		Level: logLevel(opened.cfg.Log.Level),
	}))

	// Al apagarse, el canal de avisos tiene que terminar: si no, quien lo lee
	// retiene el apagado hasta que venza su espera, y el apagado acaba en un
	// error que nadie cometió.
	closing := make(chan struct{})

	api := http.New(opened.core, log, closing)
	server := &nethttp.Server{
		Addr: fmt.Sprintf("%s:%d", opened.cfg.HTTP.Host, opened.cfg.HTTP.Port),
		// El canal de eventos permanece abierto: un tiempo de escritura
		// cortaría los avisos de las sesiones que esperan, así que solo se
		// acota la lectura de la cabecera.
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           api.Handler(),
	}

	ctx, stop := signal.NotifyContext(command.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server.RegisterOnShutdown(func() { close(closing) })

	serving := make(chan error, 1)

	go func() {
		serving <- server.ListenAndServe()
	}()

	log.Info("escuchando en http://" + server.Addr)

	select {
	case err := <-serving:
		if err != nil {
			return fmt.Errorf("sirviendo la API: %w", err)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), shutdownWait)
		defer cancel()

		if err := server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("cerrando el servidor: %w", err)
		}
	}

	return nil
}

// logLevel turns the level of the settings into the level of the registry: con
// error calla, con info dice dónde escucha y con debug cuenta las operaciones.
func logLevel(name string) slog.Level {
	switch name {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
