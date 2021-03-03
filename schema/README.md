# Schema Tool

Schema is a seed of a CLI tool that a downstream can use to use reflection and go file inspection to 
generate a base version of OpenAPI for a CRD. The resulting schema will be used by Kubernetes to
provide  

## Integration steps

### Demo

```
cd ./schema
go run ./ dump LoremIpsum | pbcopy
```

Paste this inside the CRD for LoremIpsum:

```yaml
...
      schema:
        openAPIV3Schema:
          <**paste**>
      additionalPrinterColumns:
...
```

### Downstream

Start with [example.go](./example.go), copy this into the downstream and 