class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.192"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.192/netductor-op-darwin-arm64"
      sha256 "c4b7ec6d7caf8116d46fe4c14cfa97d1213f7fb609bc55722a8c0b299a5b32e9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.192/netductor-op-darwin-amd64"
      sha256 "609636d6cd782e7a9c0c9db141c96ae3ea3d705bab71bd5f945a5071370fc285"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.192/netductor-op-linux-amd64"
      sha256 "79b7d58684d13563ce089a3d5ea3abf0ad4d8c8a3fc561ad32c6d67897f55ec9"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version")
  end
end
