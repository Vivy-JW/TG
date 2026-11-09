using System;

namespace CodeEditor.FglAnalysis
{
	// Token: 0x0200000D RID: 13
	public class FglReader
	{
		// Token: 0x0600004C RID: 76 RVA: 0x00004999 File Offset: 0x00002B99
		public FglReader(string dataToParse)
		{
			this._data = dataToParse;
			this._dataLength = this._data.Length;
			this._currPos = 0;
		}

		// Token: 0x0600004D RID: 77 RVA: 0x000049C0 File Offset: 0x00002BC0
		public char NextChar()
		{
			if (this._currPos >= this._dataLength)
			{
				return '\0';
			}
			return this._data[this._currPos++];
		}

		// Token: 0x0600004E RID: 78 RVA: 0x000049F9 File Offset: 0x00002BF9
		public char PreviousChar(int offset)
		{
			if (-2147483648 > offset || 2147483647 < offset)
			{
				this._currPos = 65535;
			}
			return this._data[this._currPos - offset];
		}

		// Token: 0x0600004F RID: 79 RVA: 0x00004A29 File Offset: 0x00002C29
		public void LookBack()
		{
			if (this._currPos > 0)
			{
				this._currPos--;
			}
		}

		// Token: 0x17000014 RID: 20
		// (get) Token: 0x06000050 RID: 80 RVA: 0x00004A42 File Offset: 0x00002C42
		public string Data
		{
			get
			{
				return this._data;
			}
		}

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x06000051 RID: 81 RVA: 0x00004A4A File Offset: 0x00002C4A
		// (set) Token: 0x06000052 RID: 82 RVA: 0x00004A52 File Offset: 0x00002C52
		public int CurrPos
		{
			get
			{
				return this._currPos;
			}
			set
			{
				this._currPos = value;
			}
		}

		// Token: 0x04000082 RID: 130
		private string _data;

		// Token: 0x04000083 RID: 131
		private int _currPos;

		// Token: 0x04000084 RID: 132
		private int _dataLength;
	}
}
