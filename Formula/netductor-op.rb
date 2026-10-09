class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.305"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.305/netductor-op-darwin-arm64"
      sha256 "a1d836b85b37b2a81d830b433345a8f0737e8e251a55416dec60815c9db355d7"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.305/netductor-op-darwin-amd64"
      sha256 "c9115d7f76f0b34bcc1eea6f2cc9b00b811e7094e58a66abff0b0927e17f51f1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.305/netductor-op-linux-amd64"
      sha256 "6270c5c84fec727213a75e3a3a0e9b45a998685d2205c2f7882afc3018a97d08"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
