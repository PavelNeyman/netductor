class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.762"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.762/netductor-op-darwin-arm64"
      sha256 "793ca229f1b8d4daeb545278fc4509f1a60b41cc8be472a19d7ca51cbf04ebb2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.762/netductor-op-darwin-amd64"
      sha256 "fcee4ceac318f53acf04d5fc36a89bd1088c3e9e2ced20d4e61696e2a6d4e40d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
