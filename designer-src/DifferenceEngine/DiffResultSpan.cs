using System;

namespace DifferenceEngine
{
	// Token: 0x0200000E RID: 14
	public class DiffResultSpan : IComparable
	{
		// Token: 0x17000006 RID: 6
		// (get) Token: 0x06000026 RID: 38 RVA: 0x00002865 File Offset: 0x00001865
		public int DestIndex
		{
			get
			{
				return this._destIndex;
			}
		}

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x06000027 RID: 39 RVA: 0x0000286D File Offset: 0x0000186D
		public int SourceIndex
		{
			get
			{
				return this._sourceIndex;
			}
		}

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x06000028 RID: 40 RVA: 0x00002875 File Offset: 0x00001875
		public int Length
		{
			get
			{
				return this._length;
			}
		}

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000029 RID: 41 RVA: 0x0000287D File Offset: 0x0000187D
		public DiffResultSpanStatus Status
		{
			get
			{
				return this._status;
			}
		}

		// Token: 0x0600002A RID: 42 RVA: 0x00002885 File Offset: 0x00001885
		protected DiffResultSpan(DiffResultSpanStatus status, int destIndex, int sourceIndex, int length)
		{
			this._status = status;
			this._destIndex = destIndex;
			this._sourceIndex = sourceIndex;
			this._length = length;
		}

		// Token: 0x0600002B RID: 43 RVA: 0x000028AA File Offset: 0x000018AA
		public static DiffResultSpan CreateNoChange(int destIndex, int sourceIndex, int length)
		{
			return new DiffResultSpan(DiffResultSpanStatus.NoChange, destIndex, sourceIndex, length);
		}

		// Token: 0x0600002C RID: 44 RVA: 0x000028B5 File Offset: 0x000018B5
		public static DiffResultSpan CreateReplace(int destIndex, int sourceIndex, int length)
		{
			return new DiffResultSpan(DiffResultSpanStatus.Replace, destIndex, sourceIndex, length);
		}

		// Token: 0x0600002D RID: 45 RVA: 0x000028C0 File Offset: 0x000018C0
		public static DiffResultSpan CreateDeleteSource(int sourceIndex, int length)
		{
			return new DiffResultSpan(DiffResultSpanStatus.DeleteSource, -1, sourceIndex, length);
		}

		// Token: 0x0600002E RID: 46 RVA: 0x000028CB File Offset: 0x000018CB
		public static DiffResultSpan CreateAddDestination(int destIndex, int length)
		{
			return new DiffResultSpan(DiffResultSpanStatus.AddDestination, destIndex, -1, length);
		}

		// Token: 0x0600002F RID: 47 RVA: 0x000028D6 File Offset: 0x000018D6
		public void AddLength(int i)
		{
			this._length += i;
		}

		// Token: 0x06000030 RID: 48 RVA: 0x000028E8 File Offset: 0x000018E8
		public override string ToString()
		{
			return string.Format("{0} (Dest: {1},Source: {2}) {3}", new object[]
			{
				this._status.ToString(),
				this._destIndex.ToString(),
				this._sourceIndex.ToString(),
				this._length.ToString()
			});
		}

		// Token: 0x06000031 RID: 49 RVA: 0x00002944 File Offset: 0x00001944
		public int CompareTo(object obj)
		{
			return this._destIndex.CompareTo(((DiffResultSpan)obj)._destIndex);
		}

		// Token: 0x04000020 RID: 32
		private const int BAD_INDEX = -1;

		// Token: 0x04000021 RID: 33
		private int _destIndex;

		// Token: 0x04000022 RID: 34
		private int _sourceIndex;

		// Token: 0x04000023 RID: 35
		private int _length;

		// Token: 0x04000024 RID: 36
		private DiffResultSpanStatus _status;
	}
}
