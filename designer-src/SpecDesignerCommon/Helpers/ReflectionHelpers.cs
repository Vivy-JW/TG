using System;
using System.ComponentModel;
using System.Reflection;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000068 RID: 104
	public static class ReflectionHelpers
	{
		// Token: 0x060003FB RID: 1019 RVA: 0x00012A6C File Offset: 0x00010C6C
		public static string GetCustomDescription(object objEnum)
		{
			FieldInfo field = objEnum.GetType().GetField(objEnum.ToString());
			DescriptionAttribute[] array = (DescriptionAttribute[])field.GetCustomAttributes(typeof(DescriptionAttribute), false);
			if (array.Length <= 0)
			{
				return objEnum.ToString();
			}
			return array[0].Description;
		}

		// Token: 0x060003FC RID: 1020 RVA: 0x00012AB7 File Offset: 0x00010CB7
		public static string Description(this Enum value)
		{
			return ReflectionHelpers.GetCustomDescription(value);
		}
	}
}
