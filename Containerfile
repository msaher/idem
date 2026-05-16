FROM alpine:latest

RUN apk update && \
    apk add --no-cache openssh sudo && \
    ssh-keygen -A

RUN sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

RUN echo "root:pass" | chpasswd

RUN adduser myuser -D \
 && echo "myuser:myuserpass" | chpasswd \
 && echo "myuser ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers
 # && echo "myuser ALL=(ALL) ALL" >> /etc/sudoers

EXPOSE 22

CMD ["/bin/sh", "-c", "/usr/sbin/sshd -D -e"]

