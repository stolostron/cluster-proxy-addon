The cluster-proxy-addon is using a customized version apiserver-network-proxy. The repo(https://github.com/stolostron/apiserver-network-proxy) is forked from the upstream repo: [apiserver-network-proxy](https://github.com/kubernetes-sigs/apiserver-network-proxy/tree/v0.1.10).

And the ANP repo is depending on the repo(https://github.com/stolostron/grpc-go/blob/master/go.mod)

If you need to update the apiserver-network-proxy version, you can follow the steps below:

### 1. git clone the forked repos
```bash
git clone git@github.com:stolostron/apiserver-network-proxy.git
git clone git@github.com:stolostron/dependency-magnet.git
```

### 2. When need to update the dependencies

When we need to upgrade the go version, the too old version of go will cause the some build jobs failed.

The commit id of the customized code can be found in:
* https://github.com/kubernetes-sigs/apiserver-network-proxy/commit/d562699c78201daef7dec97cd1847e5abffbe2ab
* https://github.com/grpc/grpc-go/commit/72dd3e65ac56e8d6b1c6a7ad25404022f1cc7a0a

First under anp project, run:

```
git fetch upstream
git branch -r
git tag -l
```

Choose the appropriate tag and branch to update. For example, tag `v0.31.1`.

```
git checkout v0.31.1
```

The find which version of grpc-go is using in the anp project, in the following file, the version of grpc-go is `v1.67.1`.

```
➜  apiserver-network-proxy git:(konnectivity-client/v0.31.1) cat go.mod | grep grpc
        google.golang.org/grpc v1.67.1
```

Then go to grpc-go project, and checkout the tag `v1.67.1`.

```
git fetch upstream
git branch -r
git tag -l
git checkout v1.67.1
```
