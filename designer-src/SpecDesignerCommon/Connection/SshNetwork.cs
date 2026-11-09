using System;
using System.Text;
using System.Windows;
using Renci.SshNet;
using Renci.SshNet.Common;
using SpecDesigner.Controls.Controls;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200005C RID: 92
	public class SshNetwork : ITTNetwork, IDisposable
	{
		// Token: 0x06000310 RID: 784 RVA: 0x0000C9DA File Offset: 0x0000ABDA
		public SshNetwork(string _ip, int _port, int _timeout)
		{
			this.ip = _ip;
			this.port = _port;
			this.timeout = _timeout;
		}

		// Token: 0x06000311 RID: 785 RVA: 0x0000CA04 File Offset: 0x0000AC04
		public bool Connect(string username, string password)
		{
			if (this.ssh == null)
			{
				this.ssh = new SshClient(this.ip, this.port, username, password);
			}
			bool flag;
			try
			{
				this.ssh.Connect();
				this.shell = this.ssh.CreateShellStream("tt", 0U, 0U, 0U, 0U, 1024);
				this.shell.DataReceived += this.shell_DataReceived;
				if (!string.IsNullOrEmpty(ConnectionInfo.This.Area))
				{
					this.Send(ConnectionInfo.This.Area);
					this.WaitFor(new string[] { ">" });
				}
				flag = true;
			}
			catch (SshAuthenticationException)
			{
				throw new Exception("帳號或密碼錯誤");
			}
			catch (SshOperationTimeoutException)
			{
				throw new Exception("主機沒有回應，請檢查連線資訊是否正確。");
			}
			catch (Exception ex)
			{
				throw ex;
			}
			return flag;
		}

		// Token: 0x06000312 RID: 786 RVA: 0x0000CAF4 File Offset: 0x0000ACF4
		private void shell_DataReceived(object sender, ShellDataEventArgs e)
		{
			this.strWorkingData = Encoding.UTF8.GetString(e.Data);
			if (this.OnDataReceived != null)
			{
				this.OnDataReceived(this, new DataReceivedEventArgs(this.strWorkingData));
			}
		}

		// Token: 0x06000313 RID: 787 RVA: 0x0000CB2C File Offset: 0x0000AD2C
		public int WaitFor(params string[] DataToWaitFor)
		{
			StringBuilder stringBuilder = new StringBuilder();
			long num = DateTime.Now.AddSeconds((double)this.timeout).Ticks;
			int i;
			for (;;)
			{
				long ticks = DateTime.Now.Ticks;
				for (i = 0; i < DataToWaitFor.Length; i++)
				{
					if (this.strWorkingData != this._prevWorkingData)
					{
						stringBuilder.Append(this.strWorkingData);
						num = DateTime.Now.AddSeconds((double)this.timeout).Ticks;
						this._prevWorkingData = this.strWorkingData;
					}
					if (stringBuilder.ToString().ToLower().Contains(DataToWaitFor[i].ToLower()))
					{
						goto Block_2;
					}
				}
				if (ticks > num)
				{
					MessageBoxResult messageBoxResult = DesignerMessageBox.Show("程式執行時間超出預期(15s)，繼續等待？", "Confirm", MessageBoxButton.YesNo);
					if (messageBoxResult != MessageBoxResult.Yes)
					{
						goto IL_0105;
					}
					num = DateTime.Now.AddSeconds((double)this.timeout).Ticks;
				}
			}
			Block_2:
			this.strWorkingData = "";
			stringBuilder.Clear();
			return i;
			IL_0105:
			this.CloseConnection();
			stringBuilder.Clear();
			this.strWorkingData = "";
			this._prevWorkingData = "";
			if (DataToWaitFor.Length > 1)
			{
				throw new TimeoutException("Timed Out when log in");
			}
			throw new TimeoutException(string.Format("Timed Out waiting for message: \"{0}\"", DataToWaitFor[0]));
		}

		// Token: 0x06000314 RID: 788 RVA: 0x0000CC84 File Offset: 0x0000AE84
		private void CloseConnection()
		{
			if (this.shell != null)
			{
				this.shell.DataReceived -= this.shell_DataReceived;
				this.shell.Close();
				this.shell.Dispose();
			}
			if (this.ssh != null)
			{
				this.ssh.Disconnect();
			}
			this.strWorkingData = "";
		}

		// Token: 0x06000315 RID: 789 RVA: 0x0000CCE4 File Offset: 0x0000AEE4
		public void Send(string msg)
		{
			this.shell.WriteLine(msg);
		}

		// Token: 0x06000316 RID: 790 RVA: 0x0000CCF2 File Offset: 0x0000AEF2
		public void SendUTF8(string msg)
		{
			this.Send(msg);
		}

		// Token: 0x170000BC RID: 188
		// (get) Token: 0x06000317 RID: 791 RVA: 0x0000CCFB File Offset: 0x0000AEFB
		public bool IsConnected
		{
			get
			{
				return this.ssh != null && this.ssh.IsConnected;
			}
		}

		// Token: 0x06000318 RID: 792 RVA: 0x0000CD12 File Offset: 0x0000AF12
		public void Close()
		{
			this.ssh.Disconnect();
		}

		// Token: 0x1400000C RID: 12
		// (add) Token: 0x06000319 RID: 793 RVA: 0x0000CD20 File Offset: 0x0000AF20
		// (remove) Token: 0x0600031A RID: 794 RVA: 0x0000CD58 File Offset: 0x0000AF58
		public event EventHandler<DataReceivedEventArgs> OnDataReceived;

		// Token: 0x0600031B RID: 795 RVA: 0x0000CD8D File Offset: 0x0000AF8D
		public void Dispose()
		{
			if (this.ssh != null)
			{
				this.ssh.Disconnect();
			}
			this.shell = null;
			this.ssh = null;
		}

		// Token: 0x170000BD RID: 189
		// (get) Token: 0x0600031C RID: 796 RVA: 0x0000CDB0 File Offset: 0x0000AFB0
		// (set) Token: 0x0600031D RID: 797 RVA: 0x0000CDB8 File Offset: 0x0000AFB8
		public string LoginPrompt { get; set; }

		// Token: 0x170000BE RID: 190
		// (get) Token: 0x0600031E RID: 798 RVA: 0x0000CDC1 File Offset: 0x0000AFC1
		// (set) Token: 0x0600031F RID: 799 RVA: 0x0000CDC9 File Offset: 0x0000AFC9
		public string PasswordPrompt { get; set; }

		// Token: 0x0400011F RID: 287
		private string ip;

		// Token: 0x04000120 RID: 288
		private int port;

		// Token: 0x04000121 RID: 289
		private int timeout;

		// Token: 0x04000122 RID: 290
		private SshClient ssh;

		// Token: 0x04000123 RID: 291
		private ShellStream shell;

		// Token: 0x04000124 RID: 292
		private string strWorkingData = "";

		// Token: 0x04000125 RID: 293
		private bool hasReceive;

		// Token: 0x04000126 RID: 294
		private string _prevWorkingData;
	}
}
