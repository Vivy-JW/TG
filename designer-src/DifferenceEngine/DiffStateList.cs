using System;

namespace DifferenceEngine
{
	// Token: 0x0200000C RID: 12
	internal class DiffStateList
	{
		// Token: 0x06000024 RID: 36 RVA: 0x00002825 File Offset: 0x00001825
		public DiffStateList(int destCount)
		{
			this._array = new DiffState[destCount];
		}

		// Token: 0x06000025 RID: 37 RVA: 0x0000283C File Offset: 0x0000183C
		public DiffState GetByIndex(int index)
		{
			DiffState diffState = this._array[index];
			if (diffState == null)
			{
				diffState = new DiffState();
				this._array[index] = diffState;
			}
			return diffState;
		}

		// Token: 0x0400001A RID: 26
		private DiffState[] _array;
	}
}
