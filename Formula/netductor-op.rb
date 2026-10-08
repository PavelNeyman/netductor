class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.279"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.279/netductor-op-darwin-arm64"
      sha256 "6cee129dbf52d29634782ff6f0de2c0a13197c6855d7579623a63e056a154379"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.279/netductor-op-darwin-amd64"
      sha256 "f9539eafe1fbf41949d559fbad18bfe635f144d4032df6edde2ec0c07f0ea03e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.279/netductor-op-linux-amd64"
      sha256 "513923b40adf79507151700111abba1f7e2ebf8f7009cd1be070c1efa819ccc2"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
