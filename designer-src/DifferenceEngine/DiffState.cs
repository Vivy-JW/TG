using System;

namespace DifferenceEngine
{
	// Token: 0x0200000B RID: 11
	internal class DiffState
	{
		// Token: 0x17000002 RID: 2
		// (get) Token: 0x0600001B RID: 27 RVA: 0x00002734 File Offset: 0x00001734
		public int StartIndex
		{
			get
			{
				return this._startIndex;
			}
		}

		// Token: 0x17000003 RID: 3
		// (get) Token: 0x0600001C RID: 28 RVA: 0x0000273C File Offset: 0x0000173C
		public int EndIndex
		{
			get
			{
				return this._startIndex + this._length - 1;
			}
		}

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x0600001D RID: 29 RVA: 0x00002750 File Offset: 0x00001750
		public int Length
		{
			get
			{
				int num;
				if (this._length > 0)
				{
					num = this._length;
				}
				else if (this._length == 0)
				{
					num = 1;
				}
				else
				{
					num = 0;
				}
				return num;
			}
		}

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x0600001E RID: 30 RVA: 0x00002780 File Offset: 0x00001780
		public DiffStatus Status
		{
			get
			{
				DiffStatus diffStatus;
				if (this._length > 0)
				{
					diffStatus = DiffStatus.Matched;
				}
				else
				{
					int length = this._length;
					if (length == -1)
					{
						diffStatus = DiffStatus.NoMatch;
					}
					else
					{
						diffStatus = DiffStatus.Unknown;
					}
				}
				return diffStatus;
			}
		}

		// Token: 0x0600001F RID: 31 RVA: 0x000027AD File Offset: 0x000017AD
		public DiffState()
		{
			this.SetToUnkown();
		}

		// Token: 0x06000020 RID: 32 RVA: 0x000027BB File Offset: 0x000017BB
		protected void SetToUnkown()
		{
			this._startIndex = -1;
			this._length = -2;
		}

		// Token: 0x06000021 RID: 33 RVA: 0x000027CC File Offset: 0x000017CC
		public void SetMatch(int start, int length)
		{
			this._startIndex = start;
			this._length = length;
		}

		// Token: 0x06000022 RID: 34 RVA: 0x000027DC File Offset: 0x000017DC
		public void SetNoMatch()
		{
			this._startIndex = -1;
			this._length = -1;
		}

		// Token: 0x06000023 RID: 35 RVA: 0x000027EC File Offset: 0x000017EC
		public bool HasValidLength(int newStart, int newEnd, int maxPossibleDestLength)
		{
			if (this._length > 0 && (maxPossibleDestLength < this._length || this._startIndex < newStart || this.EndIndex > newEnd))
			{
				this.SetToUnkown();
			}
			return this._length != -2;
		}

		// Token: 0x04000017 RID: 23
		private const int BAD_INDEX = -1;

		// Token: 0x04000018 RID: 24
		private int _startIndex;

		// Token: 0x04000019 RID: 25
		private int _length;
	}
}
