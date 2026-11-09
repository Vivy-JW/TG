using System;
using System.Collections;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Windows;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Extension;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200014A RID: 330
	public class Telnet : ITTNetwork
	{
		// Token: 0x14000024 RID: 36
		// (add) Token: 0x06000B92 RID: 2962 RVA: 0x00038CBC File Offset: 0x00036EBC
		// (remove) Token: 0x06000B93 RID: 2963 RVA: 0x00038CF4 File Offset: 0x00036EF4
		public event EventHandler<DataReceivedEventArgs> OnDataReceived;

		// Token: 0x06000B94 RID: 2964 RVA: 0x00038D2C File Offset: 0x00036F2C
		public Telnet(string Address, int Port, int CommandTimeout)
		{
			this.address = Address;
			this.port = Port;
			this.timeout = CommandTimeout;
		}

		// Token: 0x06000B95 RID: 2965 RVA: 0x00038DFC File Offset: 0x00036FFC
		public bool Connect(string username, string password)
		{
			new StringBuilder();
			IPAddress ipaddress = IPAddress.Parse(this.address);
			this.iep = new IPEndPoint(ipaddress, this.port);
			bool flag;
			try
			{
				if (!this.IsConnected)
				{
					this.s = new Socket(AddressFamily.InterNetwork, SocketType.Stream, ProtocolType.Tcp);
					this.s.SendTimeout = this.timeout * 1000;
					this.s.ReceiveTimeout = this.timeout * 1000;
					this.s.Connect(this.iep);
				}
				this.SyncWaitFor(new string[] { (this.LoginPrompt == null) ? "login:" : this.LoginPrompt });
				this.SyncSend(username);
				this.SyncWaitFor(new string[] { (this.PasswordPrompt == null) ? "password:" : this.PasswordPrompt });
				this.SyncSend(password);
				if (this.SyncWaitFor(new string[] { "incorrect", ":" }) == 0)
				{
					throw new Exception(Application.Current.FindResource("Message_LoginAccountException") as string);
				}
				if (!string.IsNullOrEmpty(ConnectionInfo.This.Area))
				{
					this.SyncSend(ConnectionInfo.This.Area);
					this.SyncWaitFor(new string[] { ">" });
				}
				flag = true;
			}
			catch (SocketException ex)
			{
				this.Close();
				throw ex;
			}
			catch (Exception ex2)
			{
				this.Close();
				throw ex2;
			}
			return flag;
		}

		// Token: 0x06000B96 RID: 2966 RVA: 0x00038FAC File Offset: 0x000371AC
		public int WaitFor(params string[] DataToWaitFor)
		{
			if (!this.IsConnected)
			{
				return 0;
			}
			StringBuilder stringBuilder = new StringBuilder();
			long num = DateTime.Now.AddSeconds((double)this.timeout).Ticks;
			this.hasReceive = false;
			int i;
			for (;;)
			{
				for (i = 0; i < DataToWaitFor.Length; i++)
				{
					if (this.hasReceive)
					{
						stringBuilder.Append(this.strWorkingData);
						num = DateTime.Now.AddSeconds((double)this.timeout).Ticks;
						this.hasReceive = false;
					}
					if (stringBuilder.IndexOf(DataToWaitFor[i], true) >= 0)
					{
						goto Block_3;
					}
				}
				long ticks = DateTime.Now.Ticks;
				if (ticks > num)
				{
					MessageBoxResult messageBoxResult = DesignerMessageBox.Show("程式執行時間超出預期(15s)，繼續等待？", "Confirm", MessageBoxButton.YesNo);
					if (messageBoxResult != MessageBoxResult.Yes)
					{
						goto IL_00F3;
					}
					num = DateTime.Now.AddSeconds((double)this.timeout).Ticks;
				}
			}
			Block_3:
			this.strWorkingData = "";
			stringBuilder.Clear();
			return i;
			IL_00F3:
			this.CloseConnection(this.s);
			stringBuilder.Clear();
			this.strWorkingData = "";
			if (DataToWaitFor.Length > 1)
			{
				throw new TimeoutException("Timed Out when log in");
			}
			throw new TimeoutException(string.Format("Timed Out waiting for message: \"{0}\"", DataToWaitFor[0]));
		}

		// Token: 0x06000B97 RID: 2967 RVA: 0x00039138 File Offset: 0x00037338
		private void DispatchMessage(string strText)
		{
			try
			{
				byte[] array = new byte[strText.Length];
				for (int i = 0; i < strText.Length; i++)
				{
					byte b = Convert.ToByte(strText[i]);
					array[i] = b;
				}
				IAsyncResult asyncResult = this.s.BeginSend(array, 0, array.Length, SocketFlags.None, delegate(IAsyncResult ar)
				{
					Socket socket = (Socket)ar.AsyncState;
					if (this.IsConnected)
					{
						AsyncCallback asyncCallback = new AsyncCallback(this.OnRecievedData);
						socket.BeginReceive(this.m_byBuff, 0, this.m_byBuff.Length, SocketFlags.None, asyncCallback, socket);
					}
				}, this.s);
				this.s.EndSend(asyncResult);
			}
			catch (Exception ex)
			{
				Console.WriteLine("出錯了,在回發數據的時候:" + ex.Message);
				this.Close();
			}
		}

		// Token: 0x06000B98 RID: 2968 RVA: 0x000391E4 File Offset: 0x000373E4
		private void OnRecievedData(IAsyncResult ar)
		{
			try
			{
				Socket socket = (Socket)ar.AsyncState;
				int num = socket.EndReceive(ar);
				if (num > 0)
				{
					string text = "";
					for (int i = 0; i < num; i++)
					{
						char c = Convert.ToChar(this.m_byBuff[i]);
						char c2 = c;
						if (c2 != '\n')
						{
							if (c2 == '\r')
							{
								text += Convert.ToString("\r\n");
							}
							else
							{
								text += Convert.ToString(c);
							}
						}
					}
					try
					{
						int length = text.Length;
						if (length == 0)
						{
							text = Convert.ToString("\r\n");
						}
						byte[] array = new byte[length];
						for (int j = 0; j < length; j++)
						{
							array[j] = Convert.ToByte(text[j]);
						}
						string text2 = this.ProcessOptions(array);
						text2 = this.ConvertToUTF8(text2);
						if (!string.IsNullOrEmpty(text2))
						{
							this.strWorkingData = text2;
							this.hasReceive = true;
							if (this.OnDataReceived != null)
							{
								this.OnDataReceived(this, new DataReceivedEventArgs(text2));
							}
						}
						this.RespondToOptions();
						goto IL_011F;
					}
					catch (Exception ex)
					{
						throw new Exception("接收數據的時候出錯了! " + ex.Message);
					}
				}
				this.CloseConnection(socket);
				IL_011F:;
			}
			catch (Exception)
			{
				this.Close();
			}
		}

		// Token: 0x06000B99 RID: 2969 RVA: 0x00039350 File Offset: 0x00037550
		private void RespondToOptions()
		{
			try
			{
				for (int i = 0; i < this.m_ListOptions.Count; i++)
				{
					string text = (string)this.m_ListOptions[i];
					this.ArrangeReply(text);
				}
				this.DispatchMessage(this.m_strResp);
				this.m_strResp = "";
				this.m_ListOptions.Clear();
			}
			catch (Exception ex)
			{
				Console.WriteLine("出錯了,在回發數據的時候 " + ex.Message);
				this.Close();
			}
		}

		// Token: 0x06000B9A RID: 2970 RVA: 0x000393E0 File Offset: 0x000375E0
		public void Send(string message)
		{
			this.DispatchMessage(message);
			this.DispatchMessage("\r\n");
		}

		// Token: 0x06000B9B RID: 2971 RVA: 0x000393F4 File Offset: 0x000375F4
		public void SendUTF8(string message)
		{
			this.DispatchUTF8Message(message);
			this.DispatchMessage("\r\n");
		}

		// Token: 0x06000B9C RID: 2972 RVA: 0x00039450 File Offset: 0x00037650
		private void DispatchUTF8Message(string message)
		{
			try
			{
				byte[] bytes = Encoding.UTF8.GetBytes(message);
				IAsyncResult asyncResult = this.s.BeginSend(bytes, 0, bytes.Length, SocketFlags.None, delegate(IAsyncResult ar)
				{
					Socket socket = (Socket)ar.AsyncState;
					if (this.IsConnected)
					{
						AsyncCallback asyncCallback = new AsyncCallback(this.OnRecievedData);
						socket.BeginReceive(this.m_byBuff, 0, this.m_byBuff.Length, SocketFlags.None, asyncCallback, socket);
					}
				}, this.s);
				this.s.EndSend(asyncResult);
			}
			catch (Exception ex)
			{
				Console.WriteLine("出錯了,在回發數據的時候:" + ex.Message);
				this.Close();
			}
		}

		// Token: 0x06000B9D RID: 2973 RVA: 0x000394D4 File Offset: 0x000376D4
		public void SyncSend(string message)
		{
			this.SyncDispatchMessage(message);
			this.SyncDispatchMessage("\r\n");
		}

		// Token: 0x06000B9E RID: 2974 RVA: 0x000394E8 File Offset: 0x000376E8
		public int SyncWaitFor(params string[] DataToWaitFor)
		{
			StringBuilder stringBuilder = new StringBuilder();
			int num = 0;
			do
			{
				num = this.s.Receive(this.m_byBuff, this.m_byBuff.Length, SocketFlags.None);
				string text = "";
				for (int i = 0; i < num; i++)
				{
					char c = Convert.ToChar(this.m_byBuff[i]);
					char c2 = c;
					if (c2 != '\n')
					{
						if (c2 == '\r')
						{
							text += Convert.ToString("\r\n");
						}
						else
						{
							text += Convert.ToString(c);
						}
					}
				}
				try
				{
					int length = text.Length;
					if (length == 0)
					{
						text = Convert.ToString("\r\n");
					}
					byte[] array = new byte[length];
					for (int j = 0; j < length; j++)
					{
						array[j] = Convert.ToByte(text[j]);
					}
					string text2 = this.ProcessOptions(array);
					text2 = this.ConvertToUTF8(text2);
					if (!string.IsNullOrEmpty(text2))
					{
						if (this.OnDataReceived != null)
						{
							this.OnDataReceived(this, new DataReceivedEventArgs(text2));
						}
						stringBuilder.Append(text2);
						for (int k = 0; k < DataToWaitFor.Length; k++)
						{
							if (stringBuilder.IndexOf(DataToWaitFor[k], true) >= 0)
							{
								this.strWorkingData = "";
								return k;
							}
						}
					}
					this.SyncRespondToOptions();
				}
				catch (Exception ex)
				{
					throw new Exception("接收數據的時候出錯了! " + ex.Message);
				}
			}
			while (num > 0);
			return -1;
		}

		// Token: 0x06000B9F RID: 2975 RVA: 0x0003965C File Offset: 0x0003785C
		private void SyncRespondToOptions()
		{
			try
			{
				for (int i = 0; i < this.m_ListOptions.Count; i++)
				{
					string text = (string)this.m_ListOptions[i];
					this.ArrangeReply(text);
				}
				this.SyncDispatchMessage(this.m_strResp);
				this.m_strResp = "";
				this.m_ListOptions.Clear();
			}
			catch (Exception ex)
			{
				Console.WriteLine("出錯了,在回發數據的時候 " + ex.Message);
				this.Close();
			}
		}

		// Token: 0x06000BA0 RID: 2976 RVA: 0x000396EC File Offset: 0x000378EC
		private void SyncDispatchMessage(string strText)
		{
			try
			{
				byte[] array = new byte[strText.Length];
				for (int i = 0; i < strText.Length; i++)
				{
					byte b = Convert.ToByte(strText[i]);
					array[i] = b;
				}
				this.s.Send(array, array.Length, SocketFlags.None);
			}
			catch (Exception ex)
			{
				Console.WriteLine("出錯了,在回發數據的時候:" + ex.Message);
				this.Close();
			}
		}

		// Token: 0x06000BA1 RID: 2977 RVA: 0x00039768 File Offset: 0x00037968
		private void ArrangeReply(string strOption)
		{
			try
			{
				bool flag = false;
				if (strOption.Length >= 3)
				{
					char c = strOption[1];
					char c2 = strOption[2];
					if (c2 == '\u0001' || c2 == '\u0003')
					{
						flag = true;
					}
					this.m_strResp += this.IAC;
					if (flag)
					{
						if (c == this.DO)
						{
							char c3 = this.WILL;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.DONT)
						{
							char c3 = this.WONT;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.WILL)
						{
							char c3 = this.DO;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.WONT)
						{
							char c3 = this.DONT;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.SB)
						{
							char c4 = strOption[3];
							if (c4 == '1')
							{
								char c3 = this.SB;
								this.m_strResp += c3;
								this.m_strResp += c2;
								this.m_strResp += '0';
								this.m_strResp += this.IAC;
								this.m_strResp += this.SE;
							}
						}
					}
					else
					{
						if (c == this.DO)
						{
							char c3 = this.WONT;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.DONT)
						{
							char c3 = this.WONT;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.WILL)
						{
							char c3 = this.DONT;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
						if (c == this.WONT)
						{
							char c3 = this.DONT;
							this.m_strResp += c3;
							this.m_strResp += c2;
						}
					}
				}
			}
			catch (Exception ex)
			{
				throw new Exception("解析參數時出錯:" + ex.Message);
			}
		}

		// Token: 0x06000BA2 RID: 2978 RVA: 0x00039A94 File Offset: 0x00037C94
		private string ProcessOptions(byte[] m_strLineToProcess)
		{
			string text = "";
			string text2 = "";
			string text3 = "";
			bool flag = false;
			try
			{
				for (int i = 0; i < m_strLineToProcess.Length; i++)
				{
					char c = Convert.ToChar(m_strLineToProcess[i]);
					text2 += Convert.ToString(c);
				}
				while (!flag)
				{
					int length = text2.Length;
					int num = text2.IndexOf(Convert.ToString(this.IAC));
					if (num > length)
					{
						num = text2.Length;
					}
					if (num != -1 && text2.Length < 3)
					{
						num = -1;
					}
					if (num != -1)
					{
						text += text2.Substring(0, num);
						char c2 = text2[num + 1];
						if (c2 == this.DO || c2 == this.DONT || c2 == this.WILL || c2 == this.WONT)
						{
							string text4 = text2.Substring(num, 3);
							this.m_ListOptions.Add(text4);
							text += text2.Substring(0, num);
							string text5 = text2.Substring(num + 3);
							text2 = text5;
						}
						else if (c2 == this.SB)
						{
							text = text2.Substring(0, num);
							int num2 = text2.IndexOf(Convert.ToString(this.SE));
							string text4 = text2.Substring(num, num2);
							this.m_ListOptions.Add(text4);
							text2 = text2.Substring(num2);
						}
						else
						{
							text = text2.Substring(0, num);
							text2 = text2.Substring(num + 1);
						}
					}
					else
					{
						text += text2;
						flag = true;
					}
				}
				text3 = text;
			}
			catch (Exception ex)
			{
				throw new Exception("解析傳入的字符串錯誤:" + ex.Message);
			}
			return text3;
		}

		// Token: 0x06000BA3 RID: 2979 RVA: 0x00039C60 File Offset: 0x00037E60
		private void CloseConnection(Socket sock)
		{
			if (sock != null)
			{
				if (this.IsConnected)
				{
					sock.Shutdown(SocketShutdown.Both);
				}
				sock.Close();
			}
			ConnectionInfo.This.IsLogin = false;
			this.strWorkingData = "";
		}

		// Token: 0x170002FF RID: 767
		// (get) Token: 0x06000BA4 RID: 2980 RVA: 0x00039C90 File Offset: 0x00037E90
		public bool IsConnected
		{
			get
			{
				if (this.s == null)
				{
					return false;
				}
				bool flag = false;
				bool blocking = this.s.Blocking;
				try
				{
					byte[] array = new byte[1];
					this.s.Send(array, 0, SocketFlags.None);
					flag = true;
				}
				catch (SocketException ex)
				{
					ex.NativeErrorCode.Equals(10035);
					flag = false;
				}
				catch (Exception)
				{
					flag = false;
				}
				return flag;
			}
		}

		// Token: 0x06000BA5 RID: 2981 RVA: 0x00039D0C File Offset: 0x00037F0C
		private string ConvertToUTF8(string str_origin)
		{
			char[] array = str_origin.ToCharArray();
			byte[] array2 = new byte[array.Length];
			for (int i = 0; i < array.Length; i++)
			{
				int num = (int)array[i];
				array2[i] = (byte)num;
			}
			Encoding encoding = Encoding.GetEncoding("utf-8");
			return encoding.GetString(array2);
		}

		// Token: 0x06000BA6 RID: 2982 RVA: 0x00039D58 File Offset: 0x00037F58
		public void Close()
		{
			this.CloseConnection(this.s);
		}

		// Token: 0x17000300 RID: 768
		// (get) Token: 0x06000BA7 RID: 2983 RVA: 0x00039D66 File Offset: 0x00037F66
		// (set) Token: 0x06000BA8 RID: 2984 RVA: 0x00039D6E File Offset: 0x00037F6E
		public string LoginPrompt { get; set; }

		// Token: 0x17000301 RID: 769
		// (get) Token: 0x06000BA9 RID: 2985 RVA: 0x00039D77 File Offset: 0x00037F77
		// (set) Token: 0x06000BAA RID: 2986 RVA: 0x00039D7F File Offset: 0x00037F7F
		public string PasswordPrompt { get; set; }

		// Token: 0x0400047E RID: 1150
		private const char IS = '0';

		// Token: 0x0400047F RID: 1151
		private const char SEND = '1';

		// Token: 0x04000480 RID: 1152
		private const char INFO = '2';

		// Token: 0x04000481 RID: 1153
		private const char VAR = '0';

		// Token: 0x04000482 RID: 1154
		private const char VALUE = '1';

		// Token: 0x04000483 RID: 1155
		private const char ESC = '2';

		// Token: 0x04000484 RID: 1156
		private const char USERVAR = '3';

		// Token: 0x04000485 RID: 1157
		private readonly char IAC = Convert.ToChar(255);

		// Token: 0x04000486 RID: 1158
		private readonly char DM = Convert.ToChar(242);

		// Token: 0x04000487 RID: 1159
		private readonly char DO = Convert.ToChar(253);

		// Token: 0x04000488 RID: 1160
		private readonly char DONT = Convert.ToChar(254);

		// Token: 0x04000489 RID: 1161
		private readonly char WILL = Convert.ToChar(251);

		// Token: 0x0400048A RID: 1162
		private readonly char WONT = Convert.ToChar(252);

		// Token: 0x0400048B RID: 1163
		private readonly char SB = Convert.ToChar(250);

		// Token: 0x0400048C RID: 1164
		private readonly char SE = Convert.ToChar(240);

		// Token: 0x0400048D RID: 1165
		private byte[] m_byBuff = new byte[100000];

		// Token: 0x0400048E RID: 1166
		private ArrayList m_ListOptions = new ArrayList();

		// Token: 0x0400048F RID: 1167
		private string m_strResp;

		// Token: 0x04000490 RID: 1168
		private Socket s;

		// Token: 0x04000491 RID: 1169
		private IPEndPoint iep;

		// Token: 0x04000492 RID: 1170
		private string address;

		// Token: 0x04000493 RID: 1171
		private int port;

		// Token: 0x04000494 RID: 1172
		private int timeout;

		// Token: 0x04000495 RID: 1173
		private string strWorkingData = "";

		// Token: 0x04000496 RID: 1174
		private bool hasReceive;
	}
}
