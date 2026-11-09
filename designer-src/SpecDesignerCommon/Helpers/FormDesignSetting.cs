using System;
using System.Collections.Generic;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000104 RID: 260
	public class FormDesignSetting
	{
		// Token: 0x17000274 RID: 628
		// (get) Token: 0x0600091A RID: 2330 RVA: 0x0002C03A File Offset: 0x0002A23A
		// (set) Token: 0x0600091B RID: 2331 RVA: 0x0002C041 File Offset: 0x0002A241
		public static int UnitWidth
		{
			get
			{
				return FormDesignSetting._unitWidth;
			}
			set
			{
				FormDesignSetting._unitWidth = value;
			}
		}

		// Token: 0x17000275 RID: 629
		// (get) Token: 0x0600091C RID: 2332 RVA: 0x0002C049 File Offset: 0x0002A249
		// (set) Token: 0x0600091D RID: 2333 RVA: 0x0002C050 File Offset: 0x0002A250
		public static int UnitHeight
		{
			get
			{
				return FormDesignSetting._unitHeight;
			}
			set
			{
				FormDesignSetting._unitHeight = value;
			}
		}

		// Token: 0x17000276 RID: 630
		// (get) Token: 0x0600091E RID: 2334 RVA: 0x0002C058 File Offset: 0x0002A258
		public static int TabelHeaderColumnHeight
		{
			get
			{
				return 24;
			}
		}

		// Token: 0x0600091F RID: 2335 RVA: 0x0002C05C File Offset: 0x0002A25C
		public static int TransformToGridWidth(double pixelWidth)
		{
			double num = pixelWidth / (double)FormDesignSetting.UnitWidth;
			return (int)Math.Ceiling(num);
		}

		// Token: 0x06000920 RID: 2336 RVA: 0x0002C07C File Offset: 0x0002A27C
		public static int TransformToGridHeight(double pixelHeight)
		{
			double num = pixelHeight / (double)FormDesignSetting.UnitHeight;
			return (int)Math.Ceiling(num);
		}

		// Token: 0x06000921 RID: 2337 RVA: 0x0002C099 File Offset: 0x0002A299
		public static int TransformToPixelWidth(double gridWidth)
		{
			return (int)(gridWidth * (double)FormDesignSetting.UnitWidth);
		}

		// Token: 0x06000922 RID: 2338 RVA: 0x0002C0A4 File Offset: 0x0002A2A4
		public static int TransformToPixelHeight(double gridHeight)
		{
			return (int)(gridHeight * (double)FormDesignSetting.UnitHeight);
		}

		// Token: 0x06000923 RID: 2339 RVA: 0x0002C0B0 File Offset: 0x0002A2B0
		public static double GetNearWidth(double pixelWidth)
		{
			double num = Math.Abs(pixelWidth) % (double)FormDesignSetting.UnitWidth;
			return pixelWidth - num;
		}

		// Token: 0x06000924 RID: 2340 RVA: 0x0002C0D0 File Offset: 0x0002A2D0
		public static double GetNearHeight(double pixelHeight)
		{
			double num = pixelHeight % (double)FormDesignSetting.UnitHeight;
			return pixelHeight - num;
		}

		// Token: 0x06000925 RID: 2341 RVA: 0x0002C0E9 File Offset: 0x0002A2E9
		public static bool IsContainer(string nodeName)
		{
			return FormDesignSetting.containers.Contains(nodeName);
		}

		// Token: 0x06000926 RID: 2342 RVA: 0x0002C0F6 File Offset: 0x0002A2F6
		public static bool IsFormField(string nodeName)
		{
			return FormDesignSetting.formFields.Contains(nodeName);
		}

		// Token: 0x06000927 RID: 2343 RVA: 0x0002C104 File Offset: 0x0002A304
		internal static bool CanSortNodes(string nodeName)
		{
			return FormDesignSetting.IsContainer(nodeName) && !(nodeName == ComponentType.Table.ToString()) && !(nodeName == ComponentType.Tree.ToString()) && !(nodeName == ComponentType.HBox.ToString()) && !(nodeName == ComponentType.VBox.ToString()) && !(nodeName == ComponentType.Page.ToString()) && !(nodeName == ComponentType.Folder.ToString());
		}

		// Token: 0x0400033B RID: 827
		private static int _unitWidth = 8;

		// Token: 0x0400033C RID: 828
		private static int _unitHeight = 20;

		// Token: 0x0400033D RID: 829
		private static List<string> containers = new List<string> { "Folder", "Form", "Grid", "Group", "HBox", "Page", "ScrollGrid", "Table", "Tree", "VBox" };

		// Token: 0x0400033E RID: 830
		private static List<string> formFields = new List<string>
		{
			"ButtonEdit", "CheckBox", "ComboBox", "DateEdit", "Edit", "FFImage", "FFLabel", "Field", "ProgressBar", "Slider",
			"SpinEdit", "TimeEdit", "TextEdit", "WebComponent", "DateTimeEdit"
		};
	}
}
