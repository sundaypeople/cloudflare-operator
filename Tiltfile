load('ext://restart_process', 'docker_build_with_restart')
load('ext://cert_manager', 'deploy_cert_manager')

def kubebuilder(DOMAIN, GROUP, VERSION, KIND, IMG='controller:latest', CONTROLLERGEN='crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases;'):

    DOCKERFILE = '''FROM golang:alpine
    WORKDIR /
    COPY ./bin/manager /manager
    CMD ["/manager"]
    '''

    def manifests():
        return 'controller-gen ' + CONTROLLERGEN

    def generate():
        return 'controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./...";'

    def vetfmt():
        return 'go vet ./...; go fmt ./...'

    def binary():
        return 'CGO_ENABLED=0 GOOS=linux GOARCH=arm GO111MODULE=on go build -o bin/manager cmd/main.go'

    installed = local("which kubebuilder")
    print("kubebuilder is present:", installed)

    DIRNAME = os.path.basename(os. getcwd())

    local_resource('make manifests', manifests(), deps=["api", "controllers"], ignore=['*/*/zz_generated.deepcopy.go'])
    local_resource('make generate', generate(), deps=["api"], ignore=['*/*/zz_generated.deepcopy.go'])

    local_resource('CRD', manifests() + 'kustomize build config/crd | kubectl apply -f -', deps=["api"], ignore=['*/*/zz_generated.deepcopy.go'])

    watch_settings(ignore=['config/crd/bases/', 'config/rbac/role.yaml', 'config/webhook/manifests.yaml'])
    k8s_yaml(kustomize('./config/dev'))

    deps = ['controllers', 'main.go']
    deps.append('api')

    local_resource('Watch&Compile', generate() + binary(), deps=deps, ignore=['*/*/zz_generated.deepcopy.go'])

    local_resource('Sample YAML', 'kubectl apply -k ./config/samples', deps=["./config/samples"], resource_deps=[DIRNAME + "-controller-manager"])

    docker_build_with_restart(IMG, '.',
     dockerfile_contents=DOCKERFILE,
     entrypoint='/manager',
     only=['./bin/manager'],
     live_update=[
           sync('./bin/manager', '/manager'),
       ]
    )

deploy_cert_manager(version="v1.6.1")
kubebuilder("cloudflare.laininthewired.github.io", "view", "v1beta1", "Cloudflare")