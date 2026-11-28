class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.785"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.785/netductor-op-darwin-arm64"
      sha256 "43d695a0f1175efed30d68cc4e591e6fbf6cffac2957032d866d8d36072c132f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.785/netductor-op-darwin-amd64"
      sha256 "a0fbea310243838edfd59ad00e35aac69044e891e0d36741e9ada9fa815dd701"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
