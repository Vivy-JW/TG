using System;
using System.ComponentModel;
using System.Text;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x02000051 RID: 81
	public class LogData : INotifyPropertyChanged, IDisposable
	{
		// Token: 0x170000A5 RID: 165
		// (get) Token: 0x060002B2 RID: 690 RVA: 0x0000BDF8 File Offset: 0x00009FF8
		public string Data
		{
			get
			{
				string text;
				try
				{
					text = this._data.ToString();
				}
				catch
				{
					text = string.Empty;
				}
				return text;
			}
		}

		// Token: 0x060002B3 RID: 691 RVA: 0x0000BE30 File Offset: 0x0000A030
		public LogData()
		{
			this._data = new StringBuilder();
		}

		// Token: 0x060002B4 RID: 692 RVA: 0x0000BE4A File Offset: 0x0000A04A
		private void SendDataChange(object sender, EventArgs e)
		{
			this.OnDataChange();
		}

		// Token: 0x060002B5 RID: 693 RVA: 0x0000BE54 File Offset: 0x0000A054
		public void Append(string content)
		{
			lock (this._data)
			{
				int num = -1;
				while ((num = content.IndexOf("\n", num + 1)) >= 0)
				{
					this.count++;
					if (this.count > LogData.MAX_LENGTH)
					{
						this._data.Clear();
						content.Remove(0, num);
						this.count = -1;
						break;
					}
				}
				this._data.Append(content);
			}
			this.OnDataChange();
		}

		// Token: 0x060002B6 RID: 694 RVA: 0x0000BEF4 File Offset: 0x0000A0F4
		public void Clear()
		{
			this.count = -1;
			this._data.Clear();
			this.OnDataChange();
		}

		// Token: 0x060002B7 RID: 695 RVA: 0x0000BF0F File Offset: 0x0000A10F
		public void Dispose()
		{
			this._data.Clear();
		}

		// Token: 0x14000008 RID: 8
		// (add) Token: 0x060002B8 RID: 696 RVA: 0x0000BF20 File Offset: 0x0000A120
		// (remove) Token: 0x060002B9 RID: 697 RVA: 0x0000BF58 File Offset: 0x0000A158
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060002BA RID: 698 RVA: 0x0000BF8D File Offset: 0x0000A18D
		private void OnDataChange()
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs("Data"));
			}
		}

		// Token: 0x04000104 RID: 260
		private StringBuilder _data;

		// Token: 0x04000105 RID: 261
		public static readonly int MAX_LENGTH = 500;

		// Token: 0x04000106 RID: 262
		private int count = -1;
	}
}
