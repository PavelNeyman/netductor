class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.735"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.735/netductor-op-darwin-arm64"
      sha256 "857bfea885c048237b8e4cef5f4baa0bb23337f726a78d9692326d0f93799d89"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.735/netductor-op-darwin-amd64"
      sha256 "022b7ab97b7134d4397d2e9abf10fdb68486e3e32770770d6ee198a3f7eca605"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.735/netductor-op-linux-amd64"
      sha256 "68244e5e3d9f15298aa261e13dbb084f4550c7b74c9f6a684d987b3139784a5b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
