# Homebrew formula for **operator** workstation binary (Mac/PC).
# VPS node binary is netductor-linux-* from GitHub Releases (installed by deploy).
class Netductor < Formula
  desc "Netductor operator (Mac): deploy, TUI, operator serve"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.0/netductor-op-darwin-arm64"
      sha256 "863efc890afa3fc51eff6f7b0c79b49a58f454b84ebece0c678164aee9232483"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.0/netductor-op-darwin-amd64"
      sha256 "47212fd7ff5ba49cfdd3bbe924aa4110bb88e8b91295b73c40edbb0376578d84"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.0/netductor-op-linux-amd64"
      sha256 "e37541155647d8a9be7754669d3090197ae012afa43908f61fc4ad97bd476418"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.0/netductor-op-linux-arm64"
      sha256 "b894296bd2e5962760e564ea9fd3d0f96389e9149ed238d738e7e3837f47bbd7"
    end
  end

  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    # Convenience symlink: historical `netductor tui` / deploy muscle memory on Mac
    bin.install_symlink "netductor-op" => "netductor"
  end

  def caveats
    <<~EOS
      Operator binary: netductor-op (also linked as netductor).
      VPS node plane is NOT this formula — deploy installs netductor-linux-* on the server.
    EOS
  end

  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
