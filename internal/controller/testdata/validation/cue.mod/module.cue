module: "validation.example"

require: {
	"k8s.io/apimachinery": "v0.27.4"
}

replace: {
	"k8s.io/apimachinery": "" @import("go")
}
