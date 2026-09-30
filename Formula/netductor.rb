class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.118"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.118/netductor-op-darwin-arm64"
      sha256 "402a9f6fa3327c33f411ca4a747d36e17e8dd0023214f04e7472483d9e6d3a9e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.118/netductor-op-darwin-amd64"
      sha256 "1eed01f876ca265525bfb1d4b09a4ba57169c47b890daf0cddf40ca2773c9f55"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.118/netductor-op-linux-amd64"
      sha256 "457fc92dd948ddd00d47b8adae6f1339ebae27155bef2757f66eec6a4d38739a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
