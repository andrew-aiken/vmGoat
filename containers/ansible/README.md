
### Volumes
- Scenario Playbook dir
- SSH
- Inventory
- config


```bash
docker run --rm --entrypoint ansible-playbook -w /mnt/ansible -v /Users/aaiken/Private/vmGoat/scenarios/test/ansible:/mnt/ansible:ro -v /Users/aaiken/Private/vmGoat/ansible.cfg:/etc/ansible/ansible.cfg:ro -v /var/folders/zm/w8zvl0jj66v4x5fp_rbtt6nh0000gn/T/vmgoat-ansible-988984707/inventory:/mnt/inventory:ro -v $HOME/.config/vmGoat/ssh:/mnt/ssh alpine/ansible:2.18.1 playbook.yaml
```
