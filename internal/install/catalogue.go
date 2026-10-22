package install

import "tasks-ai/internal/store"

// catalogue es el vocabulario con el que nace un proyecto: los ocho tipos y las
// tres marcas de forma del contrato de etiquetas, cada una con la descripción que
// el contrato le da. Sin ellas una tarea no podría llevar su tipo desde la
// primera escritura: el catálogo rechaza cualquier nombre que no esté.
func catalogue() []store.Tag {
	return []store.Tag{
		{Name: "feat", Color: store.DefaultTagColor, Description: new("Capacidad o integración nueva")},
		{Name: "fix", Color: store.DefaultTagColor, Description: new("Corrección de un defecto")},
		{Name: "docs", Color: store.DefaultTagColor, Description: new("Documentación, ADR, contrato o informe")},
		{Name: "refactor", Color: store.DefaultTagColor, Description: new("Reorganización sin cambio de comportamiento")},
		{Name: "test", Color: store.DefaultTagColor, Description: new("Pruebas, guardas y verificaciones")},
		{Name: "style", Color: store.DefaultTagColor, Description: new("Presentación visual")},
		{Name: "chore", Color: store.DefaultTagColor, Description: new("Instalación, configuración y tareas administrativas")},
		{Name: "improvement", Color: store.DefaultTagColor, Description: new("Mejora de lectura o de uso")},
		{Name: "ODD", Color: store.DefaultTagColor, Description: new("El trabajo que nace del registro ODD")},
		{Name: "epic", Color: store.DefaultTagColor, Description: new("Una tarea con subtareas")},
		{Name: "task", Color: store.DefaultTagColor, Description: new("Una tarea sin subtareas")},
	}
}
