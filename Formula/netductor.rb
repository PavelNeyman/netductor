class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.6"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.6/netductor-op-darwin-arm64"
      sha256 "e7f80e926be8482b882b4e4e7dbe61ee192390c6783605d6f828c03af2054947"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.6/netductor-op-darwin-amd64"
      sha256 "aacc3f3ee4891a2316a25f393027d7a68e10d4733073372a700782f4e3960ba6"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.6/netductor-op-linux-amd64"
      sha256 "6d915ef2e182409f8fb2e8332871d45315456fd88d75f71541c5864ae966ffdd"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.6/netductor-op-linux-arm64"
      sha256 "aa5ed73817fd80a6870046680bcfeb3694e12ec2aae2f67f497ec394a67e3064"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
