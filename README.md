# Dela
[![GitHub](https://img.shields.io/github/license/phillebaba/dela)](https://github.com/phillebaba/dela)
[![Travis (.org)](https://img.shields.io/travis/phillebaba/dela)](https://travis-ci.org/phillebaba/dela)
[![Go Report Card](https://goreportcard.com/badge/github.com/phillebaba/dela)](https://goreportcard.com/report/github.com/phillebaba/dela)
[![Docker Pulls](https://img.shields.io/docker/pulls/phillebaba/dela)](https://hub.docker.com/r/phillebaba/dela)

A solution to share secrets between namespaces in Kuberentes.

Within Kubernetes, a Pod can only read a Secret that exists within the same Namespace as itself. This limitations can create many pain points, as teams may want to share Secrets between each other, but they only have access to their own Namespaces. Dela is a alternative solution to the problem, by allowing specific Secrets to be copied actross Namespaces automatically.

## Install
Easiest way is to add a git reference in your `kustomization.yaml` file.
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- github.com/phillebaba/dela//config/default
```

Or you can add the CRD and Deploy the controller in your cluster manually.
```bash
kustomize build config/default | kubectl apply -f -
```

## Architecture
Dela uses Intent/Request model to implement it's logic. The Intent specifies which Secret should be shared in the Namespace, and the Requests asks for a copy from the Intent. This model has the benefit of allowing control of what Secrets are shared on one end, and what Secrets are copied on the other end.

<p align="center">
  <img src="./assets/overview.png">
</p>

## How To
After installing the controller create a Secret and a Intent that references the Secret in one Namespace. Note the `namespaceWhitelist` field that indicates which Namespaces are whitelisted to create a request for the intent.
```yaml
apiVersion: dela.phillebaba.io/v1alpha1
kind: Intent
metadata:
  name: main
  namespace: ns1
spec:
  secretName: main
  namespaceWhitelist:
  - ns2
---
apiVersion: v1
kind: Secret
metadata:
  name: main
  namespace: ns1
stringData:
  foo: bar
```

Then create a Request that references the Intent in another Namespace.
```yaml
apiVersion: dela.phillebaba.io/v1alpha1
kind: Request
metadata:
  name: main
  namespace: ns2
spec:
  intentRef:
    name: main
    namespace: ns1
  secretMetadata:
    name: main
```

This should now result in an identical Secret in the Second Namespace.
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: main
  namespace: ns1
stringData:
  foo: bar
```

## FAQ
**Will my Secret copy be deleted if I delete the Intent or source Secret?**
No. It could cause problems with Pods that depend on the Secret. Additionally the cat is already out of the bag so deleting the Secret would not make anything more secure. If a Secret was accidentally shared it should rather be rotated.

**Will my Secret copy be deleted if the namespace whitelist changes?**
No. See the previous answer for the reason why. The one caveat is that the Secret copy will not be updated if the source Secret changes.

**Will my Secret copy inherit any metadata?**
No. To simplify the workflow the Secret copy will not inherit any metadata. This means that the Request is required to specify the name of the Secret copy.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
