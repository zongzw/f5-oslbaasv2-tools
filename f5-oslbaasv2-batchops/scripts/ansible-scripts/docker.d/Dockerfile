# docker build . -t f5-openstack-systest-ansible-env:latest

FROM ansible/ansible-runner:latest

RUN ansible-galaxy collection install \
        ansible.netcommon:1.5.0 \
        ansible.utils:2.2.0 \
        f5networks.f5_modules:1.8.0 \
        community.mysql:2.1.0 \
        community.crypto:1.4.0 \
        community.general:2.4.0
