FROM quay.io/mfojtik/shodan:builder

ENV GOPROXY=direct
ENV GOCACHE=/tmp/cache

RUN dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo && dnf install -y gh
RUN gh config set prompt disabled && \
    gh config set git_protocol ssh

ADD bump-repo.sh /usr/bin/bump-repo.sh
