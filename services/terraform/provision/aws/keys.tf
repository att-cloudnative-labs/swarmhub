resource "tls_private_key" "node-key" {
  algorithm = "RSA"
}

resource "aws_key_pair" "rke-node-key" {
  key_name   = "rke-node-key-${var.grid_id}"
  public_key = tls_private_key.node-key.public_key_openssh
}
