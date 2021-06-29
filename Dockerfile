FROM alpine:3.13.5

ENV OPERATOR=/usr/local/bin/jenkins-operator \
    USER_UID=1001 \
    USER_NAME=jenkins-operator \
    HOME=/home/jenkins-operator

RUN apk add --no-cache ca-certificates=20191127-r2 openssh-client=8.1_p1-r0

# install operator binary
COPY jenkins-operator ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/configs /usr/local/configs

RUN chmod u+x /usr/local/bin/user_setup && \
    chmod ugo+x /usr/local/bin/entrypoint && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
