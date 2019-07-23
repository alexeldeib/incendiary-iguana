FROM alpine:3.9 as ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM v0.1.0-alpha.16 AS image
LABEL "com.github.actions.name"="Conform Action"
LABEL "com.github.actions.description"="Policy enforcement for your pipelines."
LABEL "com.github.actions.icon"="check-circle"
LABEL "com.github.actions.color"="black"
LABEL "repository"="https://github.com/talos-systems/conform.git"
LABEL "maintainer"="Andrew Rynhard <andrew@andrewrynhard.com>"
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /conform-linux-amd64 /conform
ENTRYPOINT [ "/conform" ]
CMD [ "enforce" ]