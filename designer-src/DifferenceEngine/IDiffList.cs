using System;

namespace DifferenceEngine
{
	// Token: 0x02000003 RID: 3
	public interface IDiffList
	{
		// Token: 0x06000003 RID: 3
		int Count();

		// Token: 0x06000004 RID: 4
		IComparable GetByIndex(int index);
	}
}
