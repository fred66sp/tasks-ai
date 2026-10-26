// El host aporta `@opencode/plugin` al cargar el plugin, como Pi aporta el suyo. El
// paquete publicado no expone la superficie que usa este adaptador, así que se declara
// aquí lo mínimo: `Plugin.define` recibe el id y la función de montaje, y su contexto lo
// tipa el propio adaptador en `tasks.ts`.
declare module "@opencode/plugin" {
	export const Plugin: {
		define<P>(plugin: { id: string; setup(context: P): void | Promise<void> }): unknown;
	};
}
