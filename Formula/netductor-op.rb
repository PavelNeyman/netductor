class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.659"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.659/netductor-op-darwin-arm64"
      sha256 "80e09c767c194c2235f60b19c770a89954b45357299bc9b9fdc0699a4e722f25"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.659/netductor-op-darwin-amd64"
      sha256 "fc868b127c23fa5074e74946798c02b2506d8c56f915155d8941eb78bc08a447"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
