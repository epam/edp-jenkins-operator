FROM alpine:3.18.4

ENV OPERATOR=/usr/local/bin/jenkins-operator \
    USER_UID=1001 \
    USER_NAME=jenkins-operator \
    HOME=/home/jenkins-operator

RUN apk add --no-cache ca-certificates==20230506-r0 \
                       openssh-client==9.3_p2-r0 \
                       openssl==3.1.4-r1 \
                       git==2.40.1-r0

# install operator binary
COPY ./dist/go-binary ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/configs /usr/local/configs

RUN chmod u+x /usr/local/bin/user_setup && \
    chmod ugo+x /usr/local/bin/entrypoint && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
