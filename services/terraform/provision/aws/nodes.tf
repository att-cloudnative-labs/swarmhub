locals {
  cluster_id_tag = {
    "kubernetes.io/cluster/${var.grid_id}" = "owned"
  }
}

data "aws_availability_zones" "az" {
}

resource "aws_default_subnet" "default" {
  availability_zone = data.aws_availability_zones.az.names[count.index]
  tags              = local.cluster_id_tag
  count             = length(data.aws_availability_zones.az.names)
}

resource "aws_security_group" "allow-all" {
  name        = "rke-default-security-group-${var.grid_id}"
  description = "rke"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.cluster_id_tag
}

resource "aws_instance" "rke-node-master" {
  count = 1

  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.kube_master_instance_type
  key_name               = aws_key_pair.rke-node-key.id
  iam_instance_profile   = aws_iam_instance_profile.rke-aws.name
  vpc_security_group_ids = [aws_security_group.allow-all.id]
  tags = merge(local.cluster_id_tag, map(
    "Name", "${var.grid_id}-k8s-master",
    "role", "cluster-master"
  ))
  provisioner "remote-exec" {
    connection {
      host        = coalesce(self.public_ip, self.private_ip)
      type        = "ssh"
      user        = "ubuntu"
      private_key = tls_private_key.node-key.private_key_pem
    }

    inline = [
      "curl releases.rancher.com/install-docker/19.03.sh | bash",
      "sudo groupadd docker",
      "sudo usermod -a -G docker ubuntu",
    ]
  }
}

resource "aws_instance" "rke-node-slave-locust-master" {
  count = 1

  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.master_instance_type
  key_name               = aws_key_pair.rke-node-key.id
  iam_instance_profile   = aws_iam_instance_profile.rke-aws.name
  vpc_security_group_ids = [aws_security_group.allow-all.id]
  tags = merge(local.cluster_id_tag, map(
    "Name", "${var.grid_id}-k8s-worker-locust-master",
    "role", "locust-master"
  ))

  provisioner "remote-exec" {
    connection {
      host        = coalesce(self.public_ip, self.private_ip)
      type        = "ssh"
      user        = "ubuntu"
      private_key = tls_private_key.node-key.private_key_pem
    }

    inline = [
      "curl releases.rancher.com/install-docker/19.03.sh | bash",
      "sudo groupadd docker",
      "sudo usermod -a -G docker ubuntu",
    ]
  }
}


resource "aws_instance" "rke-node-slave-locust-slave" {
  count = var.slave_instance_count

  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.slave_instance_type
  key_name               = aws_key_pair.rke-node-key.id
  iam_instance_profile   = aws_iam_instance_profile.rke-aws.name
  vpc_security_group_ids = [aws_security_group.allow-all.id]
  tags = merge(local.cluster_id_tag, map(
    "Name", "${var.grid_id}-k8s-worker-locust-slave-${count.index}",
    "role", "locust-slave"
  ))

  provisioner "remote-exec" {
    connection {
      host        = coalesce(self.public_ip, self.private_ip)
      type        = "ssh"
      user        = "ubuntu"
      private_key = tls_private_key.node-key.private_key_pem
    }

    inline = [
      "curl releases.rancher.com/install-docker/19.03.sh | bash",
      "sudo groupadd docker",
      "sudo usermod -a -G docker ubuntu",
    ]
  }
}
