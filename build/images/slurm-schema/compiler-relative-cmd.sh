#!/bin/bash

wget https://objects.githubusercontent.com/github-production-release-asset-2e65be/35699486/b4ad933a-a2d5-11e7-80b7-b4557ce42d9a?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIWNJYAX4CSVEH53A%2F20230923%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20230923T102933Z&X-Amz-Expires=300&X-Amz-Signature=7ce618b09f11d77579d8bfd720541b9536b4b975d75a086b6b84131f042966aa&X-Amz-SignedHeaders=host&actor_id=140152984&key_id=0&repo_id=35699486&response-content-disposition=attachment%3B%20filename%3Dmunge-0.5.13.tar.xz&response-content-type=application%2Foctet-stream
tar -xvf munge-0.5.13.tar.xz
cd munge && ./configure --prefix=/etc/slurm/munge && make && make install

wget https://download.schedmd.com/slurm/slurm-20.11.9.tar.bz2
tar -xvf slurm-20.11.9.tar.bz2
cd slurm && ./configure --prefix=/etc/slurm/slrum&& make && make install

