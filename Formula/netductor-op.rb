class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.656"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.656/netductor-op-darwin-arm64"
      sha256 "611ad6d5a07ba3fed8e13c7a761b799bbb93b70a25559017a4bc86e026feef8b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.656/netductor-op-darwin-amd64"
      sha256 "b93771a6c2909773b53f5d1499698d5e19fbcc1f93bce4dbb6947733ebd8fb0b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
