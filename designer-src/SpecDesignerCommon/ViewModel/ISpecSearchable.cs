using System;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200005F RID: 95
	internal interface ISpecSearchable
	{
		// Token: 0x0600032B RID: 811
		bool Contains(string tag, string value);

		// Token: 0x0600032C RID: 812
		void ChangeAttributeValue(string tag, string oldValue, string newValue);
	}
}
