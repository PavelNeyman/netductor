class Netductor < Formula
  desc "Netductor operator (Mac): deploy, TUI, operator serve"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.1"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.1/netductor-op-darwin-arm64"
      sha256 "a44343add6586a8ae2785ee57a04c29ea30cae6c6c9c437a8434a113fad027ef"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.1/netductor-op-darwin-amd64"
      sha256 "ebd40f8d52548d7487a52ac73fc84161425d8b815824f7f71ff3baf75aac836c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.1/netductor-op-linux-amd64"
      sha256 "9a3c79d9b0a98bab7b6a3ce2ab58c4ca3a444bb20102d03545bcecc8f147f351"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.1/netductor-op-linux-arm64"
      sha256 "365e397012d726380f1de2b46040fbba408c49a08eb879d0c00239eeebfeb364"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  def caveats
    <<~EOS
      Operator: netductor-op (symlink netductor). WebUI: netductor-op operator serve
      VPS node: deploy installs netductor-linux-* from Releases.
    EOS
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
