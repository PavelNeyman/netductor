class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.280"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.280/netductor-op-darwin-arm64"
      sha256 "9f2c288afde4e4a341f510dbd54340986b87bdd480f1c207ac623a54feaf56bb"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.280/netductor-op-darwin-amd64"
      sha256 "fa9582fcd1a53b39f632074b644d2ea808c391ac8bb29cf14fa5a1e9e5ad4647"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.280/netductor-op-linux-amd64"
      sha256 "8649dbae1600ef7a8ee885be8572783c345580091d4a2ac235cc497fe59c83b3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
