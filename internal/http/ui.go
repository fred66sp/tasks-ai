package http

import (
	"io/fs"
	nethttp "net/http"
	"tasks-ai/web"
)

// appPage es la página de la interfaz dentro de lo incrustado: la raíz del
// servidor lleva hasta ella, y de ahí en adelante cada ruta que la página
// nombra —sus guiones, sus estilos y el Vue vendorizado— sale de la misma
// carpeta.
const appPage = "/task-manager/app-vue.html"

// interfaceFiles serves the embedded interface as loose files, without cache:
// es una herramienta de desarrollo y un bundle viejo escondido en la caché
// haría pensar que un cambio no llegó.
func interfaceFiles() nethttp.Handler {
	files, err := fs.Sub(web.Files, "public")
	if err != nil {
		// El patrón va incrustado en el binario: si no está, el binario está
		// roto y no hay nada que servir.
		panic("la interfaz no viaja en el binario: " + err.Error())
	}

	return noStore(nethttp.FileServerFS(files))
}

// noStore writes the header of a response that must not be kept.
func noStore(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// interfaceIndex carries whoever opens the root of the server to the page of
// the interface: la API conserva sus rutas /api/... y la raíz es de la
// interfaz (ADR-010).
func interfaceIndex(w nethttp.ResponseWriter, r *nethttp.Request) {
	nethttp.Redirect(w, r, appPage, nethttp.StatusFound)
}
