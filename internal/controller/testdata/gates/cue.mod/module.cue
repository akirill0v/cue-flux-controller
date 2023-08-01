module: "app.example"

require: {
	"k8s.io/api": "v0.27.4"
}

require: {
	"k8s.io/apimachinery": "v0.27.4" @indirect()
}

replace: {
	"k8s.io/api":          "" @import("go")
	"k8s.io/apimachinery": "" @import("go")
}
