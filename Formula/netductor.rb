class Netductor < Formula
  desc "Netductor operator (Mac client): deploy + local admin UI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.2"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.2/netductor-op-darwin-arm64"
      sha256 "d6d3423ab5a7d7544890b5dbcf504099e16eb82144c52a7247610b4ccc4c5a0e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.2/netductor-op-darwin-amd64"
      sha256 "cdae21af0023f1981af247895493a08c2f193b510ab47a267b4fc12ebd3796ac"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.2/netductor-op-linux-amd64"
      sha256 "90faeefa0db9b9c699f6c52e6c7a555c39f1c1588486a6296c5286a9d150fb5f"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.2/netductor-op-linux-arm64"
      sha256 "abc841f0be418283e92c732ec405929d85fcd5811822acf9279ad6095b5829ae"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  def caveats
    <<~EOS
      Mac client: netductor-op operator serve  → http://127.0.0.1:7373/
      Node API tunnel: netductor-op tunnel --host PRIMARY
      VPS has no product web UI — API only.
    EOS
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
